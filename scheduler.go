package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
)

// runReportScheduler periodically runs due report schedules and delivers each
// via the notifications service (email/SMS/WhatsApp per the schedule's channel).
//
// DueReportSchedules claims and advances each schedule in a single atomic
// statement, so a schedule is handed to exactly one caller even when several
// replicas of this service tick at the same moment. That is what keeps a
// recipient from being sent the same report once per running instance.
func runReportScheduler(ctx context.Context, repo *store.Repository, bus *events.Bus) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, s := range repo.DueReportSchedules() {
				run := repo.RunReport(models.ReportRunInput{
					TemplateID: s.TemplateID,
					Name:       s.Name,
					EmailTo:    s.Recipient,
				})
				channel := s.Channel
				if channel == "" {
					channel = "email"
				}
				// Each notification template exists on one channel only, so the
				// template follows the channel. An older schedule stored with a
				// channel that has no report template is skipped, loudly,
				// rather than failing at the notifications service unseen.
				var template string
				switch channel {
				case "email":
					template = "dms.alert"
				case "whatsapp":
					template = "dms.alert.whatsapp"
				case "in_app":
					template = "alert-in-app"
				default:
					slog.WarnContext(ctx, "scheduled report not delivered: no template for channel",
						"schedule", s.ID, "channel", channel)
					continue
				}
				body := fmt.Sprintf("Your scheduled report \"%s\" generated %d rows.", run.Name, run.RowCount)
				if bus != nil && bus.Enabled() && s.Recipient != "" {
					bus.PublishAlert(ctx, channel, s.Recipient, template, map[string]string{
						"Title":   "Scheduled report: " + run.Name,
						"Body":    body,
						"Message": body, // alert-in-app renders Message
					}, run.JobID)
				}
				slog.InfoContext(ctx, "scheduled report delivered", "schedule", s.ID, "recipient", s.Recipient, "channel", channel)
			}
		}
	}
}
