package presenter

import (
	"encoding/json"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/dto/room"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// ToRoomResponse converts a Room entity to RoomResponse DTO
func ToRoomResponse(r *entities.Room) *room.RoomResponse {
	if r == nil {
		return nil
	}

	// Parse settings from JSON
	var settings map[string]interface{}
	if r.Settings != nil {
		json.Unmarshal(r.Settings, &settings)
	}

	response := &room.RoomResponse{
		ID:                  r.ID.String(),
		Name:                r.Name,
		Description:         r.Description,
		Slug:                r.Slug,
		HostID:              r.HostID.String(),
		Type:                string(r.Type),
		Status:              string(r.Status),
		LivekitRoomName:     r.LivekitRoomName,
		MaxParticipants:     r.MaxParticipants,
		CurrentParticipants: r.CurrentParticipants,
		Settings:            settings,
		ScheduledStartTime:  r.ScheduledStartTime,
		ScheduledEndTime:    r.ScheduledEndTime,
		StartedAt:           r.StartedAt,
		EndedAt:             r.EndedAt,
		Duration:            r.Duration,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}

	// Include host if loaded
	if r.Host != nil {
		response.Host = ToUserResponse(r.Host)
	}

	return response
}

// ToRoomListResponse converts a slice of Room entities to RoomListResponse
func ToRoomListResponse(rooms []*entities.Room, total int64, page, pageSize int) *room.RoomListResponse {
	roomResponses := make([]*room.RoomResponse, len(rooms))
	for i, r := range rooms {
		roomResponses[i] = ToRoomResponse(r)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return &room.RoomListResponse{
		Rooms:      roomResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// ToParticipantResponse converts a Participant entity to ParticipantResponse DTO
func ToParticipantResponse(p *entities.Participant) *room.ParticipantResponse {
	if p == nil {
		return nil
	}

	response := &room.ParticipantResponse{
		ID:                p.ID.String(),
		RoomID:            p.RoomID.String(),
		Role:              string(p.Role),
		Status:            string(p.Status),
		InvitedEmail:      p.InvitedEmail,
		InvitedAt:         p.InvitedAt,
		JoinedAt:          p.JoinedAt,
		LeftAt:            p.LeftAt,
		Duration:          p.Duration,
		CanShareScreen:    p.CanShareScreen,
		CanRecord:         p.CanRecord,
		CanMuteOthers:     p.CanMuteOthers,
		IsMuted:           p.IsMuted,
		IsHandRaised:      p.IsHandRaised,
		ConnectionQuality: p.ConnectionQuality,
		CreatedAt:         p.CreatedAt,
	}

	// UserID might be nil for invited participants
	if p.UserID != nil {
		userIDStr := p.UserID.String()
		response.UserID = userIDStr
	}

	// InvitedBy might be nil
	if p.InvitedBy != nil {
		invitedByStr := p.InvitedBy.String()
		response.InvitedBy = &invitedByStr
	}

	// Include user if loaded
	if p.User != nil {
		response.User = ToUserResponse(p.User)
	}

	return response
}

// ToParticipantListResponse converts a slice of Participant entities to ParticipantListResponse
func ToParticipantListResponse(participants []*entities.Participant) *room.ParticipantListResponse {
	participantResponses := make([]*room.ParticipantResponse, len(participants))
	for i, p := range participants {
		participantResponses[i] = ToParticipantResponse(p)
	}

	return &room.ParticipantListResponse{
		Participants: participantResponses,
		Total:        len(participants),
	}
}

// ToMyInvitationsResponse converts participant invitations to MyInvitationsResponse
func ToMyInvitationsResponse(participants []*entities.Participant) *room.MyInvitationsResponse {
	invitations := make([]room.InvitationItem, 0, len(participants))

	for _, p := range participants {
		if p.Room == nil {
			continue
		}

		item := room.InvitationItem{
			ParticipantID: p.ID.String(),
			RoomID:        p.RoomID.String(),
			RoomName:      p.Room.Name,
			RoomType:      string(p.Room.Type),
			InvitedAt:     *p.InvitedAt,
		}

		// Add inviter info if available
		if p.InvitedBy != nil {
			item.InviterID = p.InvitedBy.String()
		}

		// Get inviter name from User relation if loaded
		if p.User != nil {
			item.InviterName = p.User.Name
		}

		invitations = append(invitations, item)
	}

	return &room.MyInvitationsResponse{
		Invitations: invitations,
		Total:       len(invitations),
	}
}
