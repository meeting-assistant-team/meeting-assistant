package entities

import "time"

type ActionItem struct {
    ID                 string    `json:"id"`
    RoomID             string    `json:"room_id"`
    SummaryID          string    `json:"summary_id"`
    AssignedTo         string    `json:"assigned_to"`
    CreatedBy          string    `json:"created_by"`
    Title              string    `json:"title"`
    Description        string    `json:"description"`
    Type               string    `json:"type"`
    Priority           string    `json:"priority"`
    Status             string    `json:"status"`
    DueDate            *time.Time `json:"due_date,omitempty"`
    TranscriptReference string   `json:"transcript_reference"`
    TimestampInMeeting int       `json:"timestamp_in_meeting"`
    ClickupTaskID      string    `json:"clickup_task_id"`
    ClickupURL         string    `json:"clickup_url"`
    CreatedAt          time.Time `json:"created_at"`
}
