package repositories

import "github.com/johnquangdev/meeting-assistant/internal/domain/entities"

// AIRepository defines persistence operations for transcripts and AI results
type AIRepository interface {
	// Transcripts
	SaveTranscript(t *entities.Transcript) error
	GetTranscriptByRecordingID(recordingID string) (*entities.Transcript, error)

	// Summaries
	SaveMeetingSummary(s *entities.MeetingSummary) error
	GetMeetingSummaryByRoom(roomID string) (*entities.MeetingSummary, error)

	// Action items
	SaveActionItems(items []*entities.ActionItem) error
	ListActionItemsByRoom(roomID string) ([]*entities.ActionItem, error)

	// Participant reports
	SaveParticipantReport(r *entities.ParticipantReport) error
	GetParticipantReportsByRoom(roomID string) ([]*entities.ParticipantReport, error)

	// Jobs
	SaveAIJob(meetingID, jobID, status string) error
}
