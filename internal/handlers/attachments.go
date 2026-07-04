package handlers

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func apierrService(c *gin.Context, msg string) {
	apierr.JSONStatus(c, http.StatusInternalServerError, msg)
}

func apierrStatus(c *gin.Context, status int, msg string) {
	apierr.JSONStatus(c, status, msg)
}

const maxUploadBytes = 25 << 20 // 25 MiB

// attachmentURL builds the gateway-reachable download URL for an attachment id.
func (h *API) attachmentURL(id string) string {
	return h.Cfg.PublicAPIURL + h.Cfg.GatewayAPIPrefix + "/v1/attachments/" + id + "/download"
}

// storeUpload reads the multipart "file" field, writes the bytes to object
// storage and persists an attachment row. Shared by the generic upload route
// and the execution-photo route.
func (h *API) storeUpload(c *gin.Context, ownerType, ownerID string) (models.Attachment, bool) {
	if h.Storage == nil {
		apierrService(c, "attachment storage is not configured")
		return models.Attachment{}, false
	}
	fh, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "multipart field 'file' is required")
		return models.Attachment{}, false
	}
	if fh.Size > maxUploadBytes {
		apierrStatus(c, http.StatusRequestEntityTooLarge, "file exceeds 25MB limit")
		return models.Attachment{}, false
	}
	src, err := fh.Open()
	if err != nil {
		apierrService(c, "cannot read upload")
		return models.Attachment{}, false
	}
	defer src.Close()

	filename := path.Base(fh.Filename)
	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	key := "att/" + uuid.NewString() + "/" + sanitizeName(filename)
	size, err := h.Storage.Put(key, contentType, src)
	if err != nil {
		apierrService(c, "storage write failed")
		return models.Attachment{}, false
	}
	a, err := h.Repo.CreateAttachment(models.Attachment{
		OwnerType:   ownerType,
		OwnerID:     ownerID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		StorageKey:  key,
		UploadedBy:  auth.ActorName(c),
	})
	if err != nil {
		_ = h.Storage.Delete(key)
		apierrService(c, "attachment record failed")
		return models.Attachment{}, false
	}
	a.URL = h.attachmentURL(a.ID)
	return a, true
}

func (h *API) UploadAttachment(c *gin.Context) {
	ownerType := c.DefaultPostForm("ownerType", c.Query("ownerType"))
	ownerID := c.DefaultPostForm("ownerId", c.Query("ownerId"))
	a, ok := h.storeUpload(c, ownerType, ownerID)
	if !ok {
		return
	}
	h.recordAudit(c, "UploadAttachment", store.AuditDetail("attachment", a.ID, "uploaded"))
	c.JSON(http.StatusCreated, a)
}

func (h *API) ListAttachments(c *gin.Context) {
	items := h.Repo.ListAttachments(c.Query("ownerType"), c.Query("ownerId"))
	for i := range items {
		items[i].URL = h.attachmentURL(items[i].ID)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "data": items})
}

// ListExecutionPhotos returns attachments linked to a retail-execution task.
func (h *API) ListExecutionPhotos(c *gin.Context) {
	items := h.Repo.ListAttachments("execution", c.Param("id"))
	for i := range items {
		items[i].URL = h.attachmentURL(items[i].ID)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "data": items})
}

func (h *API) DownloadAttachment(c *gin.Context) {
	a, err := h.Repo.GetAttachment(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	if h.Storage == nil {
		apierrService(c, "attachment storage is not configured")
		return
	}
	rc, err := h.Storage.Open(a.StorageKey)
	if err != nil {
		notFound(c)
		return
	}
	defer rc.Close()
	c.Header("Content-Disposition", "inline; filename=\""+a.Filename+"\"")
	c.DataFromReader(http.StatusOK, a.Size, a.ContentType, rc, nil)
}

// UploadExecutionPhoto attaches a photo to a retail-execution task.
func (h *API) UploadExecutionPhoto(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.storeUpload(c, "execution", id)
	if !ok {
		return
	}
	h.publish(c, "dms.execution.photo_added", gin.H{"executionId": id, "attachmentId": a.ID})
	h.recordAudit(c, "UploadExecutionPhoto", store.AuditDetail("execution", id, "photo added"))
	c.JSON(http.StatusCreated, a)
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimSpace(name)
	if name == "" {
		return "file"
	}
	return name
}
