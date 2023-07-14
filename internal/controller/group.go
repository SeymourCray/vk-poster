package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
	"vk-poster/internal/entity"
	"vk-poster/internal/usecase/webapi"
)

type groupsRoutes struct {
	g GroupsUseCase
}

func NewGroupsRoutes(r *gin.RouterGroup, groups GroupsUseCase) {
	h := groupsRoutes{g: groups}

	groupsRouters := r.Group("/groups")
	{
		groupsRouters.GET("", h.groups)
		groupsRouters.POST("/new", h.addGroup)
		groupsRouters.POST("/change-downloading-limit", h.changeDownloadingLimit)
		groupRouters := groupsRouters.Group("/:groupID")
		{
			groupRouters.GET("", h.group)
			groupRouters.POST("/update", h.updateGroup)
			groupRouters.POST("/remove", h.removeGroup)
			groupRouters.POST("/start", h.startGroup)
			groupRouters.POST("/break", h.breakGroup)

			sourcesRouters := groupRouters.Group("/sources")
			{
				sourcesRouters.POST("/new", h.addSource)
				sourceRouters := sourcesRouters.Group("/:sourceID")
				{
					sourceRouters.GET("", h.source)
					sourceRouters.POST("/update", h.updateSource)
					sourceRouters.POST("/remove", h.removeSource)
				}
			}

			scheduleRouters := groupRouters.Group("/schedule")
			{
				scheduleRouters.POST("/new", h.addEvent)
				eventRouters := scheduleRouters.Group("/:eventID")
				{
					eventRouters.POST("/remove", h.removeEvent)
				}
			}
		}
	}
}

func (r *groupsRoutes) groups(c *gin.Context) {
	groups, err := r.g.GetGroups()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.HTML(http.StatusOK, "groups.tmpl", gin.H{
		"Groups":           groups,
		"DownloadingLimit": webapi.MaxDownloadingTime.Seconds(),
	})
}

func (r *groupsRoutes) addGroup(c *gin.Context) {
	var group entity.Group
	groupID, err := r.g.InsertGroup(group)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) group(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	group, sources, events, err := r.g.GetGroup(groupID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	var photoSources, videoSources []entity.Source
	for _, src := range sources {
		switch src.Category {
		case entity.Video:
			videoSources = append(videoSources, src)
		case entity.Photo:
			photoSources = append(photoSources, src)
		}
	}

	lastScan := group.LastScanTime.Format(time.DateTime)
	if group.LastScanTime.IsZero() {
		lastScan = "----------"
	}

	scanTime := group.ScanTime.Format(entity.LayoutTime)

	isRunning := r.g.GetGroupStatus(groupID)

	c.HTML(http.StatusOK, "edit-group.tmpl", gin.H{
		"Group":        group,
		"PhotoSources": photoSources,
		"VideoSources": videoSources,
		"ScanTime":     scanTime,
		"Events":       events,
		"LastScan":     lastScan,
		"IsRunning":    isRunning,
	})
}

func (r *groupsRoutes) updateGroup(c *gin.Context) {
	var group entity.Group

	t := c.PostForm("scan-time")
	scanTime, _ := time.ParseInLocation(entity.LayoutTime, t, time.Local)

	if err := c.Bind(&group); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)
	group.Id = groupID
	group.ScanTime = scanTime

	if err := r.g.UpdateGroup(group); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) removeGroup(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	if err := r.g.DeleteGroup(groupID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, "/private/groups")
}

func (r *groupsRoutes) startGroup(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	r.g.StartGroup(groupID)

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) breakGroup(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	if err := r.g.BreakGroup(groupID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) addSource(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	var source entity.Source
	sourceID, err := r.g.InsertSource(source, groupID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d/sources/%d", groupID, sourceID))
}

func (r *groupsRoutes) source(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)
	param = c.Param("sourceID")
	sourceID, _ := strconv.Atoi(param)

	source, err := r.g.GetSource(groupID, sourceID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.HTML(http.StatusOK, "edit-source.tmpl", gin.H{
		"Source":  source,
		"GroupID": groupID,
	})
}

func (r *groupsRoutes) removeSource(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)
	param = c.Param("sourceID")
	sourceID, _ := strconv.Atoi(param)

	err := r.g.DeleteSource(sourceID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) updateSource(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)
	param = c.Param("sourceID")
	sourceID, _ := strconv.Atoi(param)

	var source entity.Source
	if err := c.Bind(&source); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	source.Id = sourceID

	if err := r.g.UpdateSource(source); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) addEvent(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)

	var event entity.Event

	t := c.PostForm("publication-datetime")
	dateTime, _ := time.ParseInLocation(entity.LayoutDateTime, t, time.Local)

	if err := c.Bind(&event); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	event.Datetime = dateTime

	if err := r.g.InsertEvent(event, groupID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) removeEvent(c *gin.Context) {
	param := c.Param("groupID")
	groupID, _ := strconv.Atoi(param)
	param = c.Param("eventID")
	eventID, _ := strconv.Atoi(param)

	if err := r.g.DeleteEvent(eventID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/private/groups/%d", groupID))
}

func (r *groupsRoutes) changeDownloadingLimit(c *gin.Context) {
	l := c.PostForm("downloading-limit")
	newLimit, _ := strconv.Atoi(l)

	webapi.MaxDownloadingTime = time.Duration(newLimit) * time.Second

	c.Redirect(http.StatusSeeOther, "/private/groups")
}
