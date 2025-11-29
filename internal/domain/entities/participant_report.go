package entities

import "time"

type ParticipantReport struct {
	ID                string                 `json:"id"`
	RoomID            string                 `json:"room_id"`
	ParticipantID     string                 `json:"participant_id"`
	SummaryID         string                 `json:"summary_id"`
	ReportContent     string                 `json:"report_content"`
	SpeakingTime      int                    `json:"speaking_time"`
	SpeakingPercent   float64                `json:"speaking_percentage"`
	ContributionCount int                    `json:"contribution_count"`
	QuestionsAsked    int                    `json:"questions_asked"`
	Metrics           map[string]interface{} `json:"metrics"`
	CreatedAt         time.Time              `json:"created_at"`
}
