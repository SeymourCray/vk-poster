package controller

import (
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/gin-gonic/gin"
	"vk-poster/config"
	"vk-poster/internal/entity"
)

type GroupsUseCase interface {
	AddVkProvider(client *api.VK)
	GetGroups() ([]entity.Group, error)
	InsertGroup(group entity.Group) (int, error)
	GetGroup(groupId int) (entity.Group, []entity.Source, []entity.Event, error)
	UpdateGroup(group entity.Group) error
	StartGroup(groupId int)
	BreakGroup(groupId int) error
	GetSource(groupId int, sourceId int) (entity.Source, error)
	InsertSource(source entity.Source, groupId int) (int, error)
	DeleteSource(sourceId int) error
	UpdateSource(source entity.Source) error
	InsertEvent(event entity.Event, groupId int) error
	DeleteEvent(eventId int) error
	DeleteGroup(groupId int) error
	GetGroupStatus(groupId int) bool
}

func NewRouter(r *gin.Engine, groups GroupsUseCase, cfg config.Config) {
	NewAuthRoutes(r, groups, cfg)
}
