package repo

import (
	"github.com/jmoiron/sqlx"
	"vk-poster/internal/entity"
)

const (
	getGroupsQuery          = `select id, name, description, tag, link, stopwords, n_days, scan_time, last_scan_time from groups;`
	getGroupQuery           = `select * from groups where id = $1;`
	deleteGroupQuery        = `delete from groups where id = $1;`
	insertGroupQuery        = `insert into groups(name, description, tag, link, stopwords, n_days, scan_time, last_scan_time) values($1,$2,$3,$4,$5,$6,$7,$8) returning id;`
	updateGroupQuery        = `update groups set name = :name, description = :description, tag = :tag, link = :link, stopwords = :stopwords, n_days = :n_days, scan_time = :scan_time where id =:id;`
	updateLastScanTimeQuery = `update groups set  last_scan_time = :last_scan_time where id =:id;`
	getSourcesQuery         = `select id, category, link, duration_limit, like_limit, comment_limit, repost_limit, view_limit from sources where group_id = $1;`
	getSourceQuery          = `select id, category, link, duration_limit, like_limit, comment_limit, repost_limit, view_limit from sources where group_id = $1 and id = $2;`
	deleteSourceQuery       = `delete from sources where id = $1;`
	deleteGroupSourceQuery  = `delete from sources where group_id = $1;`
	insertSourceQuery       = `insert into sources(category, link, duration_limit, like_limit, comment_limit, repost_limit, view_limit, group_id) values($1,$2,$3,$4,$5,$6,$7,$8) returning id;`
	updateSourceQuery       = `update sources set category = $2, link = $3, duration_limit = $4, like_limit = $5, comment_limit = $6, repost_limit = $7, view_limit = $8 where id = $1;`
	getEventsQuery          = `select id, category, date_time, repeat_interval from events where group_id = $1;`
	deleteEventQuery        = `delete from events where id = $1;`
	deleteGroupEventQuery   = `delete from events where group_id = $1;`
	insertEventQuery        = `insert into events(category, date_time, repeat_interval, group_id) values($1,$2,$3,$4);`
)

type GroupsRepo struct {
	connection *sqlx.DB
}

func NewGroupsRepository(connection *sqlx.DB) *GroupsRepo {
	return &GroupsRepo{connection: connection}
}

func (r *GroupsRepo) GetGroups() ([]entity.Group, error) {
	var groups []entity.Group
	err := r.connection.Select(&groups, getGroupsQuery)
	return groups, err
}

func (r *GroupsRepo) GetGroup(groupID int) (entity.Group, error) {
	var group entity.Group
	err := r.connection.Get(&group, getGroupQuery, groupID)
	return group, err
}

func (r *GroupsRepo) DeleteGroup(groupID int) error {
	tx, _ := r.connection.Begin()

	tx.Exec(deleteGroupSourceQuery, groupID)
	tx.Exec(deleteGroupEventQuery, groupID)
	tx.Exec(deleteGroupQuery, groupID)

	err := tx.Commit()

	return err
}

func (r *GroupsRepo) UpdateLastScanTime(group entity.Group) error {
	_, err := r.connection.NamedExec(updateLastScanTimeQuery, group)
	return err
}

func (r *GroupsRepo) InsertGroup(group entity.Group) (int, error) {
	res := r.connection.QueryRow(
		insertGroupQuery,
		group.Name,
		group.Description,
		group.Tag,
		group.Link,
		group.Stopwords,
		group.NDays,
		group.ScanTime,
		group.LastScanTime,
	)

	var groupID int
	err := res.Scan(&groupID)
	return groupID, err
}

func (r *GroupsRepo) UpdateGroup(group entity.Group) error {
	_, err := r.connection.NamedExec(updateGroupQuery, group)
	return err
}

func (r *GroupsRepo) GetSources(groupID int) ([]entity.Source, error) {
	var sources []entity.Source
	err := r.connection.Select(&sources, getSourcesQuery, groupID)
	return sources, err
}

func (r *GroupsRepo) GetSource(groupID, sourceID int) (entity.Source, error) {
	var source entity.Source
	err := r.connection.Get(&source, getSourceQuery, groupID, sourceID)
	return source, err
}

func (r *GroupsRepo) DeleteSource(sourceID int) error {
	_, err := r.connection.Exec(deleteSourceQuery, sourceID)
	return err
}

func (r *GroupsRepo) InsertSource(source entity.Source, groupID int) (int, error) {
	res := r.connection.QueryRow(
		insertSourceQuery,
		source.Category,
		source.Link,
		source.DurationLimit,
		source.LikeLimit,
		source.CommentLimit,
		source.RepostLimit,
		source.ViewLimit,
		groupID,
	)

	var sourceID int
	err := res.Scan(&sourceID)
	return sourceID, err
}

func (r *GroupsRepo) UpdateSource(source entity.Source) error {
	_, err := r.connection.Exec(
		updateSourceQuery,
		source.Id,
		source.Category,
		source.Link,
		source.DurationLimit,
		source.LikeLimit,
		source.CommentLimit,
		source.RepostLimit,
		source.ViewLimit,
	)

	return err
}

func (r *GroupsRepo) GetEvents(groupID int) ([]entity.Event, error) {
	var events []entity.Event
	err := r.connection.Select(&events, getEventsQuery, groupID)
	return events, err
}

func (r *GroupsRepo) DeleteEvent(eventID int) error {
	_, err := r.connection.Exec(deleteEventQuery, eventID)
	return err
}

func (r *GroupsRepo) InsertEvent(event entity.Event, groupID int) error {
	_, err := r.connection.Exec(
		insertEventQuery,
		event.Category,
		event.Datetime,
		event.RepeatInterval,
		groupID,
	)

	return err
}
