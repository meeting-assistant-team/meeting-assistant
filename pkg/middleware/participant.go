package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	roomUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/room"
	"github.com/labstack/echo/v4"
)

// RequireHostRole middleware: only allow host to perform action
func RequireHostRole(roomService roomUsecase.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			roomID, err := uuid.Parse(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":   "invalid_room_id",
					"message": "room ID must be a valid UUID",
				})
			}
			userID, ok := c.Get("user_id").(uuid.UUID)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error":   "unauthorized",
					"message": "user not authenticated",
				})
			}
			room, err := roomService.GetRoom(c.Request().Context(), roomID)
			if err != nil {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error":   "room_not_found",
					"message": err.Error(),
				})
			}
			if room.HostID != userID {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "not_host",
					"message": "user is not the host",
				})
			}
			return next(c)
		}
	}
}

// RequireParticipantStatus middleware: only allow participant with certain status
func RequireParticipantStatus(roomService roomUsecase.Service, allowedStatus ...entities.ParticipantStatus) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			roomID, err := uuid.Parse(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":   "invalid_room_id",
					"message": "room ID must be a valid UUID",
				})
			}
			userID, ok := c.Get("user_id").(uuid.UUID)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error":   "unauthorized",
					"message": "user not authenticated",
				})
			}
			participant, err := roomService.GetParticipantByRoomAndUser(c.Request().Context(), roomID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "not_participant",
					"message": "user is not a participant in this room",
				})
			}
			for _, status := range allowedStatus {
				if participant.Status == status {
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":   "invalid_participant_status",
				"message": "participant status not allowed for this action",
			})
		}
	}
}
