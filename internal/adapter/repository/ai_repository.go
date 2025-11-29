package repository

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	repo "github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

type aiRepository struct {
	db *gorm.DB
}

// NewAIRepository creates a new AI repository backed by GORM
func NewAIRepository(db *gorm.DB) repo.AIRepository {
	return &aiRepository{db: db}
}

func (r *aiRepository) SaveTranscript(t *entities.Transcript) error {
	// marshal JSONB fields
	segments, _ := json.Marshal(t.Segments)
	words, _ := json.Marshal(t.Words)
	metadata, _ := json.Marshal(map[string]interface{}{})

	// Upsert by recording_id
	q := `INSERT INTO transcripts (id, recording_id, room_id, text, language, segments, words, confidence_score, has_speakers, speaker_count, processing_time, model_used, metadata, created_at)
        VALUES (?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?, ?, ?, ?, ?, ?::jsonb, ?)
        ON CONFLICT (recording_id) DO UPDATE SET text = EXCLUDED.text, segments = EXCLUDED.segments, words = EXCLUDED.words, confidence_score = EXCLUDED.confidence_score, has_speakers = EXCLUDED.has_speakers, speaker_count = EXCLUDED.speaker_count, processing_time = EXCLUDED.processing_time, model_used = EXCLUDED.model_used, updated_at = NOW()`

	return r.db.Exec(q, t.ID, t.RecordingID, t.RoomID, t.Text, t.Language, string(segments), string(words), t.ConfidenceScore, t.HasSpeakers, t.SpeakerCount, t.ProcessingTime, t.ModelUsed, string(metadata), time.Now()).Error
}

func (r *aiRepository) GetTranscriptByRecordingID(recordingID string) (*entities.Transcript, error) {
	row := r.db.Raw(`SELECT id, recording_id, room_id, text, language, segments::text AS segments, words::text AS words, confidence_score, has_speakers, speaker_count, processing_time, model_used, created_at FROM transcripts WHERE recording_id = ? LIMIT 1`, recordingID).Row()
	var res struct {
		ID              string
		RecordingID     string
		RoomID          string
		Text            string
		Language        string
		Segments        string
		Words           string
		ConfidenceScore *float64
		HasSpeakers     bool
		SpeakerCount    int
		ProcessingTime  *int
		ModelUsed       string
		CreatedAt       time.Time
	}
	if err := row.Scan(&res.ID, &res.RecordingID, &res.RoomID, &res.Text, &res.Language, &res.Segments, &res.Words, &res.ConfidenceScore, &res.HasSpeakers, &res.SpeakerCount, &res.ProcessingTime, &res.ModelUsed, &res.CreatedAt); err != nil {
		return nil, err
	}

	var segments []entities.Segment
	var words []entities.WordTimestamp
	if res.Segments != "" {
		_ = json.Unmarshal([]byte(res.Segments), &segments)
	}
	if res.Words != "" {
		_ = json.Unmarshal([]byte(res.Words), &words)
	}

	t := &entities.Transcript{
		ID:           res.ID,
		RecordingID:  res.RecordingID,
		RoomID:       res.RoomID,
		Text:         res.Text,
		Language:     res.Language,
		Segments:     segments,
		Words:        words,
		HasSpeakers:  res.HasSpeakers,
		SpeakerCount: res.SpeakerCount,
		ModelUsed:    res.ModelUsed,
		CreatedAt:    res.CreatedAt,
	}
	if res.ConfidenceScore != nil {
		t.ConfidenceScore = *res.ConfidenceScore
	}
	if res.ProcessingTime != nil {
		t.ProcessingTime = *res.ProcessingTime
	}

	return t, nil
}

func (r *aiRepository) SaveMeetingSummary(s *entities.MeetingSummary) error {
	keyPoints, _ := json.Marshal(s.KeyPoints)
	decisions, _ := json.Marshal(s.Decisions)
	topics, _ := json.Marshal(s.Topics)
	openQ, _ := json.Marshal(s.OpenQuestions)
	nextSteps, _ := json.Marshal(s.NextSteps)
	metadata, _ := json.Marshal(map[string]interface{}{})

	q := `INSERT INTO meeting_summaries (id, room_id, transcript_id, executive_summary, key_points, decisions, topics, open_questions, next_steps, overall_sentiment, sentiment_breakdown, model_used, processing_time, metadata, created_at)
        VALUES (?, ?, ?, ?, ?::jsonb, ?::jsonb, ?::jsonb, ?::jsonb, ?::jsonb, ?, '{}'::jsonb, ?, ?, ?::jsonb, ?)
        ON CONFLICT (room_id) DO UPDATE SET executive_summary = EXCLUDED.executive_summary, key_points = EXCLUDED.key_points, decisions = EXCLUDED.decisions, topics = EXCLUDED.topics, open_questions = EXCLUDED.open_questions, next_steps = EXCLUDED.next_steps, overall_sentiment = EXCLUDED.overall_sentiment, processing_time = EXCLUDED.processing_time, updated_at = NOW()`

	return r.db.Exec(q, s.ID, s.RoomID, s.TranscriptID, s.ExecutiveSummary, string(keyPoints), string(decisions), string(topics), string(openQ), string(nextSteps), s.OverallSentiment, s.ModelUsed, s.ProcessingTime, string(metadata), time.Now()).Error
}

func (r *aiRepository) GetMeetingSummaryByRoom(roomID string) (*entities.MeetingSummary, error) {
	row := r.db.Raw(`SELECT id, room_id, transcript_id, executive_summary, key_points::text AS key_points, decisions::text AS decisions, topics::text AS topics, open_questions::text AS open_questions, next_steps::text AS next_steps, overall_sentiment, processing_time, model_used, created_at FROM meeting_summaries WHERE room_id = ? LIMIT 1`, roomID).Row()
	var res struct {
		ID               string
		RoomID           string
		TranscriptID     string
		ExecutiveSummary string
		KeyPoints        string
		Decisions        string
		Topics           string
		OpenQuestions    string
		NextSteps        string
		OverallSentiment *float64
		ProcessingTime   *int
		ModelUsed        string
		CreatedAt        time.Time
	}
	if err := row.Scan(&res.ID, &res.RoomID, &res.TranscriptID, &res.ExecutiveSummary, &res.KeyPoints, &res.Decisions, &res.Topics, &res.OpenQuestions, &res.NextSteps, &res.OverallSentiment, &res.ProcessingTime, &res.ModelUsed, &res.CreatedAt); err != nil {
		return nil, err
	}

	var keyPoints []string
	var decisions []map[string]string
	var topics []string
	var openQ []string
	var next []string
	_ = json.Unmarshal([]byte(res.KeyPoints), &keyPoints)
	_ = json.Unmarshal([]byte(res.Decisions), &decisions)
	_ = json.Unmarshal([]byte(res.Topics), &topics)
	_ = json.Unmarshal([]byte(res.OpenQuestions), &openQ)
	_ = json.Unmarshal([]byte(res.NextSteps), &next)

	ms := &entities.MeetingSummary{
		ID:               res.ID,
		RoomID:           res.RoomID,
		TranscriptID:     res.TranscriptID,
		ExecutiveSummary: res.ExecutiveSummary,
		KeyPoints:        keyPoints,
		Decisions:        decisions,
		Topics:           topics,
		OpenQuestions:    openQ,
		NextSteps:        next,
		ModelUsed:        res.ModelUsed,
		CreatedAt:        res.CreatedAt,
	}
	if res.OverallSentiment != nil {
		ms.OverallSentiment = *res.OverallSentiment
	}
	if res.ProcessingTime != nil {
		ms.ProcessingTime = *res.ProcessingTime
	}
	return ms, nil
}

func (r *aiRepository) SaveActionItems(items []*entities.ActionItem) error {
	for _, it := range items {
		// Basic insert
		q := `INSERT INTO action_items (id, room_id, summary_id, assigned_to, created_by, title, description, type, priority, status, due_date, transcript_reference, timestamp_in_meeting, clickup_task_id, clickup_url, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, assigned_to = EXCLUDED.assigned_to, clickup_task_id = EXCLUDED.clickup_task_id, clickup_url = EXCLUDED.clickup_url, updated_at = NOW()`
		if err := r.db.Exec(q, it.ID, it.RoomID, it.SummaryID, it.AssignedTo, it.CreatedBy, it.Title, it.Description, it.Type, it.Priority, it.Status, it.DueDate, it.TranscriptReference, it.TimestampInMeeting, it.ClickupTaskID, it.ClickupURL, time.Now()).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *aiRepository) ListActionItemsByRoom(roomID string) ([]*entities.ActionItem, error) {
	rows, err := r.db.Raw(`SELECT id, room_id, summary_id, assigned_to, created_by, title, description, type, priority, status, due_date, transcript_reference, timestamp_in_meeting, clickup_task_id, clickup_url, created_at FROM action_items WHERE room_id = ?`, roomID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*entities.ActionItem
	for rows.Next() {
		var it entities.ActionItem
		var dueDate *time.Time
		if err := rows.Scan(&it.ID, &it.RoomID, &it.SummaryID, &it.AssignedTo, &it.CreatedBy, &it.Title, &it.Description, &it.Type, &it.Priority, &it.Status, &dueDate, &it.TranscriptReference, &it.TimestampInMeeting, &it.ClickupTaskID, &it.ClickupURL, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.DueDate = dueDate
		items = append(items, &it)
	}
	return items, nil
}

func (r *aiRepository) SaveParticipantReport(rp *entities.ParticipantReport) error {
	// Upsert unique (room_id, participant_id)
	q := `INSERT INTO participant_reports (id, room_id, participant_id, summary_id, report_content, speaking_time, speaking_percentage, contribution_count, questions_asked, metrics, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?) ON CONFLICT (room_id, participant_id) DO UPDATE SET report_content = EXCLUDED.report_content, speaking_time = EXCLUDED.speaking_time, speaking_percentage = EXCLUDED.speaking_percentage, contribution_count = EXCLUDED.contribution_count, questions_asked = EXCLUDED.questions_asked, metrics = EXCLUDED.metrics, updated_at = NOW()`
	metrics, _ := json.Marshal(rp.Metrics)
	return r.db.Exec(q, rp.ID, rp.RoomID, rp.ParticipantID, rp.SummaryID, rp.ReportContent, rp.SpeakingTime, rp.SpeakingPercent, rp.ContributionCount, rp.QuestionsAsked, string(metrics), time.Now()).Error
}

func (r *aiRepository) GetParticipantReportsByRoom(roomID string) ([]*entities.ParticipantReport, error) {
	rows, err := r.db.Raw(`SELECT id, room_id, participant_id, summary_id, report_content, speaking_time, speaking_percentage, contribution_count, questions_asked, metrics::text AS metrics, created_at FROM participant_reports WHERE room_id = ?`, roomID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.ParticipantReport
	for rows.Next() {
		var rp entities.ParticipantReport
		var metricsStr string
		if err := rows.Scan(&rp.ID, &rp.RoomID, &rp.ParticipantID, &rp.SummaryID, &rp.ReportContent, &rp.SpeakingTime, &rp.SpeakingPercent, &rp.ContributionCount, &rp.QuestionsAsked, &metricsStr, &rp.CreatedAt); err != nil {
			return nil, err
		}
		var metrics map[string]interface{}
		_ = json.Unmarshal([]byte(metricsStr), &metrics)
		rp.Metrics = metrics
		out = append(out, &rp)
	}
	return out, nil
}

func (r *aiRepository) SaveAIJob(meetingID, jobID, status string) error {
	// Best-effort: update latest recording for the room (meetingID) with processing times
	// Find latest recording by room_id
	var recID string
	row := r.db.Raw(`SELECT id FROM recordings WHERE room_id = ? ORDER BY created_at DESC LIMIT 1`, meetingID).Row()
	if err := row.Scan(&recID); err != nil {
		// no recording found - ignore
		// still try to insert ai_jobs record if jobID present
		if jobID == "" {
			return nil
		}
		// insert ai_jobs best-effort (table may not exist)
		qins := `INSERT INTO ai_jobs (id, recording_id, room_id, status, attempts, last_error, created_at) VALUES (?, ?, ?, ?, 0, NULL, ?)`
		return r.db.Exec(qins, jobID, recID, meetingID, status, time.Now()).Error
	}
	switch status {
	case "processing":
		// update recordings state
		_ = r.db.Exec(`UPDATE recordings SET status = 'processing', processing_started_at = NOW() WHERE id = ?`, recID).Error
		// insert ai_jobs
		qins := `INSERT INTO ai_jobs (id, recording_id, room_id, status, attempts, last_error, created_at) VALUES (?, ?, ?, ?, 0, NULL, ?)
            ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`
		return r.db.Exec(qins, jobID, recID, meetingID, status, time.Now()).Error
	case "completed":
		_ = r.db.Exec(`UPDATE recordings SET status = 'completed', processing_completed_at = NOW() WHERE id = ?`, recID).Error
		qins := `INSERT INTO ai_jobs (id, recording_id, room_id, status, attempts, last_error, created_at) VALUES (?, ?, ?, ?, 0, NULL, ?)
            ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`
		return r.db.Exec(qins, jobID, recID, meetingID, status, time.Now()).Error
	case "failed":
		_ = r.db.Exec(`UPDATE recordings SET status = 'failed', processing_completed_at = NOW() WHERE id = ?`, recID).Error
		qins := `INSERT INTO ai_jobs (id, recording_id, room_id, status, attempts, last_error, created_at) VALUES (?, ?, ?, ?, 0, NULL, ?)
            ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`
		return r.db.Exec(qins, jobID, recID, meetingID, status, time.Now()).Error
	default:
		return nil
	}
}
