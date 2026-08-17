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
				// WhatsApp needs a whatsapp-channel template; all other channels use
				// the generic email/text alert template.
				template := "dms.alert"
				if channel == "whatsapp" {
					template = "dms.alert.whatsapp"
				}
				if bus != nil && bus.Enabled() && s.Recipient != "" {
					bus.PublishAlert(ctx, channel, s.Recipient, template, map[string]string{
						"Title": "Scheduled report: " + run.Name,
						"Body":  fmt.Sprintf("Your scheduled report \"%s\" generated %d rows.", run.Name, run.RowCount),
					}, run.JobID)
				}
				slog.InfoContext(ctx, "scheduled report delivered", "schedule", s.ID, "recipient", s.Recipient, "channel", channel)
			}
		}
	}
}
