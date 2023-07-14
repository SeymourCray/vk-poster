package usecase

import (
	"errors"
	"fmt"
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"sync"
	"time"
	"vk-poster/internal/entity"
	"vk-poster/internal/usecase/webapi"
)

type GroupsRepo interface {
	GetGroups() ([]entity.Group, error)
	GetGroup(groupID int) (entity.Group, error)
	DeleteGroup(groupID int) error
	UpdateLastScanTime(group entity.Group) error
	InsertGroup(group entity.Group) (int, error)
	UpdateGroup(group entity.Group) error
	GetSources(groupID int) ([]entity.Source, error)
	GetSource(groupID, sourceID int) (entity.Source, error)
	DeleteSource(sourceID int) error
	InsertSource(source entity.Source, groupID int) (int, error)
	UpdateSource(source entity.Source) error
	GetEvents(groupID int) ([]entity.Event, error)
	DeleteEvent(eventID int) error
	InsertEvent(event entity.Event, groupID int) error
}

type VkProvider interface {
	GetPosts(source entity.Source) ([]entity.Post, error)
	PutPost(post entity.Post, groupName, groupTag, groupLink string, publishDate int64) error
}

type SyncMap struct {
	sync.RWMutex
	m map[int]bool
}

func NewSyncMap() *SyncMap {
	return &SyncMap{
		RWMutex: sync.RWMutex{},
		m:       make(map[int]bool),
	}
}

func (syncMap *SyncMap) Get(k int) (bool, bool) {
	syncMap.RLock()
	defer syncMap.RUnlock()
	v, ok := syncMap.m[k]
	return v, ok
}

func (syncMap *SyncMap) Set(k int, v bool) {
	syncMap.Lock()
	defer syncMap.Unlock()
	syncMap.m[k] = v
}

type GroupsUseCase struct {
	repo       GroupsRepo
	vkProvider VkProvider
	states     *SyncMap
}

func NewGroupsUseCase(groupsRepo GroupsRepo) *GroupsUseCase {
	return &GroupsUseCase{
		repo:   groupsRepo,
		states: NewSyncMap(),
	}
}

func (useCase *GroupsUseCase) AddVkProvider(client *api.VK) {
	useCase.vkProvider = webapi.NewVkProvider(client)
}

func (useCase *GroupsUseCase) GetGroups() ([]entity.Group, error) {
	return useCase.repo.GetGroups()
}

func (useCase *GroupsUseCase) InsertGroup(group entity.Group) (int, error) {
	return useCase.repo.InsertGroup(group)
}

func (useCase *GroupsUseCase) GetGroup(groupId int) (entity.Group, []entity.Source, []entity.Event, error) {
	group, err := useCase.repo.GetGroup(groupId)
	if err != nil {
		return entity.Group{}, []entity.Source{}, []entity.Event{}, err
	}

	sources, err := useCase.repo.GetSources(groupId)
	if err != nil {
		return entity.Group{}, []entity.Source{}, []entity.Event{}, err
	}

	events, err := useCase.repo.GetEvents(groupId)
	if err != nil {
		return entity.Group{}, []entity.Source{}, []entity.Event{}, err
	}

	return group, sources, events, nil
}

func (useCase *GroupsUseCase) UpdateGroup(group entity.Group) error {
	err := useCase.repo.UpdateGroup(group)
	return err
}

func (useCase *GroupsUseCase) StartGroup(groupId int) {
	go useCase.StartScanLoop(groupId)
}

func (useCase *GroupsUseCase) DeleteGroup(groupId int) error {
	if err := useCase.repo.DeleteGroup(groupId); err != nil {
		return err
	}

	useCase.states.Set(groupId, false)

	return nil
}

func (useCase *GroupsUseCase) BreakGroup(groupId int) error {
	if v, ok := useCase.states.Get(groupId); v && ok {
		useCase.states.Set(groupId, false)

		return nil
	}

	return fmt.Errorf("group %d has already been stopped", groupId)
}

func (useCase *GroupsUseCase) GetGroupStatus(groupId int) bool {
	v, _ := useCase.states.Get(groupId)
	return v
}

func (useCase *GroupsUseCase) GetSource(groupId, sourceId int) (entity.Source, error) {
	return useCase.repo.GetSource(groupId, sourceId)
}

func (useCase *GroupsUseCase) InsertSource(source entity.Source, groupId int) (int, error) {
	return useCase.repo.InsertSource(source, groupId)
}

func (useCase *GroupsUseCase) DeleteSource(sourceId int) error {
	return useCase.repo.DeleteSource(sourceId)
}

func (useCase *GroupsUseCase) UpdateSource(source entity.Source) error {
	return useCase.repo.UpdateSource(source)
}

func (useCase *GroupsUseCase) InsertEvent(event entity.Event, groupId int) error {
	return useCase.repo.InsertEvent(event, groupId)
}

func (useCase *GroupsUseCase) DeleteEvent(eventId int) error {
	return useCase.repo.DeleteEvent(eventId)
}

func (useCase *GroupsUseCase) StartAllScanLoops() error {
	groups, err := useCase.repo.GetGroups()
	if err != nil {
		return err
	}

	for _, group := range groups {
		go useCase.StartScanLoop(group.Id)
	}

	return nil
}

func (useCase *GroupsUseCase) StartScanLoop(groupId int) {
	group, err := useCase.repo.GetGroup(groupId)
	if err != nil {
		logrus.Errorf("error occurred during starting scan loop for group %s: cannot get current state of group from db: %s", group.Name, err)
		return
	}

	if v, ok := useCase.states.Get(groupId); v && ok {
		logrus.Warnf("StartScanLoop (group %s): ScanLoop is already running", group.Name)
		return
	}

	useCase.states.Set(groupId, true)

	logrus.Infof("StartScanLoop (group %s): StartScanLoop is wating for the next scan period", group.Name)

	select {
	case <-time.After(time.Until(group.LastScanTime.AddDate(0, 0, group.NDays))):
		useCase.ScanLoop(groupId)
	}
}

func (useCase *GroupsUseCase) ScanLoop(groupId int) {
	// get current state of group
	group, err := useCase.repo.GetGroup(groupId)
	if err != nil {
		logrus.Errorf("ScanLoop (group id %d): error occurred during scan loop: cannot get current state of group from db: %s", groupId, err)
		useCase.states.Set(groupId, false)
		return
	}

	logrus.Infof("ScanLoop (group %s): ScanLoop is starting", group.Name)

	// make first launch of scan period immediate
	group.ScanTime = FixDatetimeDate(group.ScanTime)
	if group.ScanTime.Before(time.Now()) {
		group.ScanTime = group.ScanTime.AddDate(0, 0, 1)
	}
	group.ScanTime = group.ScanTime.AddDate(0, 0, -group.NDays)

	for {
		select {
		case <-time.After(time.Until(group.ScanTime.AddDate(0, 0, group.NDays))):
			// prepare necessary variables
			var stopwords []string
			var sources []entity.Source
			var events []entity.Event
			var posts []entity.Post
			var postsSlicesWithPhoto, postsSlicesWithVideo [][]entity.Post
			var postsWithPhoto, postsWithVideo []entity.Post

			// get data from db
			if v, ok := useCase.states.Get(groupId); v && ok {
				logrus.Infof("ScanLoop (group %s): ScanLoop is starting scan period", group.Name)

				// get current state of group
				group, err = useCase.repo.GetGroup(groupId)
				if err != nil {
					logrus.Errorf("ScanLoop (group %s): error occurred during scan loop: cannot get current state of group from db: %s", group.Name, err)
					useCase.states.Set(groupId, false)
					return
				}

				sources, err = useCase.repo.GetSources(groupId)
				if err != nil {
					logrus.Errorf("ScanLoop (group %s): error occurred during scan loop: cannot get current sources of group from db: %s", group.Name, err)
					useCase.states.Set(groupId, false)
					return
				}
				if len(sources) == 0 {
					logrus.Warnf("ScanLoop (group %s): sources are empty, finishing ScanLoop", group.Name)
					useCase.states.Set(groupId, false)
					return
				}

				events, err = useCase.repo.GetEvents(groupId)
				if err != nil {
					logrus.Errorf("ScanLoop (group %s): error occurred during scan loop: cannot get current events of group from db: %s", group.Name, err)
					useCase.states.Set(groupId, false)
					return
				}
				if len(events) == 0 {
					logrus.Warnf("ScanLoop (group %s): events are empty, finishing ScanLoop", group.Name)
					useCase.states.Set(groupId, false)
					return
				}

				if useCase.vkProvider == nil {
					logrus.Warn("ScanLoop: no token in service")
					logrus.Warn("ScanLoop: please, go to /private/token")
					useCase.states.Set(groupId, false)
					return
				}

				// make sure time.Time variables' date part are actual
				group.ScanTime = FixDatetimeDate(group.ScanTime)

				stopwords = strings.Split(group.Stopwords, ",")
			} else {
				return
			}

			// download, filter and sort posts
			if v, ok := useCase.states.Get(groupId); v && ok {
				// load posts from sources
				for _, source := range sources {
					logrus.Infof("ScanLoop (group %s): downloading posts (source link %s)", group.Name, source.Link)

					posts, err = useCase.vkProvider.GetPosts(source)
					if err != nil {
						logrus.Warnf("ScanLoop (group %s): error occured during downloading posts (source link %s): %s", group.Name, source.Link, err)
						logrus.Infof("ScanLoop (group %s): restarting scan loop", group.Name)
						useCase.states.Set(groupId, false)
						go useCase.StartScanLoop(groupId)
						return
					}

					// filter and sort posts according to source limits
					filteredPosts := FilterPosts(posts, source, group.NDays, stopwords, group.ScanTime)
					sortedPosts := SortPosts(filteredPosts)

					switch source.Category {
					case entity.Photo:
						postsSlicesWithPhoto = append(postsSlicesWithPhoto, sortedPosts)
						break
					case entity.Video:
						postsSlicesWithVideo = append(postsSlicesWithVideo, sortedPosts)
						break
					}

					time.Sleep(time.Second)
				}
			} else {
				return
			}

			// flatten posts 2D slices into 1D slice
			postsWithPhoto = FlattenPostsSlices(postsSlicesWithPhoto)
			postsWithVideo = FlattenPostsSlices(postsSlicesWithVideo)

			// make postponed posts
			if v, ok := useCase.states.Get(groupId); v && ok {
				// make scheduled posts
				var iPhoto, iVideo int
				for _, event := range events {
					logrus.Infof("ScanLoop (group %s): starting processing posting (event id %d)", group.Name, event.Id)

					// make sure time.Time variables' date part are actual
					event.Datetime = FixDatetimeDate(event.Datetime)

					// if it's too late to start posting today, start posting tomorrow
					postTime := event.Datetime
					if postTime.Before(time.Now()) {
						postTime = postTime.AddDate(0, 0, 1)
					}

					// make scheduled post as long as scanTime < postTime < scanTime+NDays
					var post entity.Post
					for ; postTime.Before(group.ScanTime.AddDate(0, 0, group.NDays)); postTime = postTime.AddDate(0, 0, event.RepeatInterval) {

						switch event.Category {
						case entity.Photo:
							if iPhoto >= len(postsWithPhoto) {
								logrus.Infof("ScanLoop (group %s): there is no any available posts with photo, skipping post loop", group.Name)
								continue
							} else {
								post = postsWithPhoto[iPhoto]
								iPhoto++
							}
						case entity.Video:
							if iVideo >= len(postsWithVideo) {
								logrus.Infof("ScanLoop (group %s): there is no any available posts with video, skipping post loop", group.Name)
								continue
							} else {
								post = postsWithVideo[iVideo]
								iVideo++
							}
						}

						logrus.Infof("ScanLoop (group %s): making post (event id %d, post time %v), post info: id=%d, fromId=%d", group.Name, event.Id, postTime, post.Id, post.FromId)
						err = useCase.vkProvider.PutPost(post, group.Name, group.Tag, group.Link, postTime.Unix())

						initPostTime := postTime
					PostingLoop:
						for err != nil {
							switch errors.Unwrap(err) {
							case webapi.VkError:
								logrus.Errorf("ScanLoop (group %s): VK error occurred during scan loop: cannot put post on vk: %s", group.Name, err)
								postTime = postTime.Add(time.Hour)
								logrus.Infof("ScanLoop (group %s): increasing publish time by 1 hour, new time: %v", group.Name, postTime)

							case webapi.InternalError:
								logrus.Errorf("ScanLoop (group %s): internal error occurred during scan loop: cannot put post on vk: %s", group.Name, err)
								logrus.Infof("ScanLoop (group %s): trying to skip post and get next one", group.Name)

								switch event.Category {
								case entity.Photo:
									if iPhoto >= len(postsWithPhoto) {
										logrus.Infof("ScanLoop (group %s): there is no any available posts with photo, skipping post loop", group.Name)
										break PostingLoop
									} else {
										post = postsWithPhoto[iPhoto]
										iPhoto++
									}
								case entity.Video:
									if iVideo >= len(postsWithVideo) {
										logrus.Infof("ScanLoop (group %s): there is no any available posts with video, skipping post loop", group.Name)
										break PostingLoop
									} else {
										post = postsWithVideo[iVideo]
										iVideo++
									}
								}
							}

							logrus.Infof("ScanLoop (group %s): attempting to make post again (event id %d, post time %v), post info: id=%d, fromId=%d", group.Name, event.Id, postTime, post.Id, post.FromId)
							err = useCase.vkProvider.PutPost(post, group.Name, group.Tag, group.Link, postTime.Unix())
						}
						postTime = initPostTime

						//TODO
						time.Sleep(time.Second)
					}
				}
			} else {
				return
			}

			// update group last scan time
			group.LastScanTime = group.ScanTime
			if err = useCase.repo.UpdateLastScanTime(group); err != nil {
				logrus.Errorf("ScanLoop (group %s): error occurred during scan loop: cannot update group last scan time: %s", group.Name, err)
				useCase.states.Set(groupId, false)
				return
			}
		}
	}
}

func FixDatetimeDate(datetime time.Time) time.Time {
	currentYear, currentMonth, currentDay := time.Now().Date()
	datetimeYear, datetimeMonth, datetimeDay := datetime.Date()
	return datetime.AddDate(currentYear-datetimeYear, int(currentMonth)-int(datetimeMonth), currentDay-datetimeDay)
}

func FilterPosts(posts []entity.Post, source entity.Source, nDays int, stopwords []string, scanTime time.Time) []entity.Post {
	var filteredPosts []entity.Post
	var containsStopwords bool
	for _, post := range posts {

		containsStopwords = false
		for _, stopword := range stopwords {
			if stopword != "" && strings.Contains(post.Text, stopword) {
				containsStopwords = true
			}
		}

		if scanTime.Sub(post.Date) <= time.Hour*time.Duration(24*nDays) &&
			!containsStopwords &&
			post.Likes >= source.LikeLimit &&
			post.Views >= source.ViewLimit &&
			post.Reposts >= source.RepostLimit &&
			post.Comments >= source.CommentLimit {
			switch source.Category {
			case entity.Photo:
				filteredPosts = append(filteredPosts, post)
				break
			case entity.Video:
				isCorrectVideo := true
				for _, attachment := range post.Attachments {
					if attachment.Category == entity.Video && attachment.Duration < source.DurationLimit {
						isCorrectVideo = false
						break
					}
				}

				if isCorrectVideo {
					filteredPosts = append(filteredPosts, post)
				}
				break
			}
		}
	}

	return filteredPosts
}

func SortPosts(posts []entity.Post) []entity.Post {
	scoresTable := make(map[int]float32)

	likesSlice := make([]entity.Post, len(posts))
	copy(likesSlice, posts)
	sort.Slice(likesSlice, func(i, j int) bool {
		return likesSlice[i].Likes > likesSlice[j].Likes
	})
	for i, post := range likesSlice {
		scoresTable[post.Id] += float32(i + 1)
	}

	viewsSlice := make([]entity.Post, len(posts))
	copy(viewsSlice, posts)
	sort.Slice(viewsSlice, func(i, j int) bool {
		return viewsSlice[i].Views > viewsSlice[j].Views
	})
	for i, post := range viewsSlice {
		scoresTable[post.Id] += float32(i + 1)
	}

	repostsSlice := make([]entity.Post, len(posts))
	copy(repostsSlice, posts)
	sort.Slice(repostsSlice, func(i, j int) bool {
		return repostsSlice[i].Reposts > repostsSlice[j].Reposts
	})
	for i, post := range repostsSlice {
		scoresTable[post.Id] += float32(i + 1)
	}

	commentsSlice := make([]entity.Post, len(posts))
	copy(commentsSlice, posts)
	sort.Slice(commentsSlice, func(i, j int) bool {
		return commentsSlice[i].Comments > commentsSlice[j].Comments
	})
	for i, post := range commentsSlice {
		scoresTable[post.Id] += float32(i + 1)
	}

	sort.Slice(posts, func(i, j int) bool {
		return scoresTable[posts[i].Id] < scoresTable[posts[j].Id]
	})

	return posts
}

func FlattenPostsSlices(postsSlices [][]entity.Post) []entity.Post {
	var flattenedSlice []entity.Post

	var maxLength int
	for _, postsSlice := range postsSlices {
		if len(postsSlice) > maxLength {
			maxLength = len(postsSlice)
		}
	}

	for postIndex := 0; postIndex < maxLength; postIndex++ {
		for sliceIndex := range postsSlices {
			if postIndex < len(postsSlices[sliceIndex]) && postsSlices[sliceIndex] != nil && postIndex < len(postsSlices[sliceIndex]) {
				flattenedSlice = append(flattenedSlice, postsSlices[sliceIndex][postIndex])
			}
		}
	}

	return flattenedSlice
}
