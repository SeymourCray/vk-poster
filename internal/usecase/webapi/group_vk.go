package webapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/SevereCloud/vksdk/v2/api"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/wader/goutubedl"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vk-poster/internal/entity"
)

const (
	getPostsAmount = 50
	filesPath      = "data/files/"
	delay          = time.Second
)

var (
	InternalError      = errors.New("internal error")
	VkError            = errors.New("vk error")
	MaxDownloadingTime = time.Minute * 10
)

var attachmentType = map[string]entity.Category{
	"video": entity.Video,
	"photo": entity.Photo,
}

type VkProvider struct {
	client *api.VK
}

func NewVkProvider(client *api.VK) *VkProvider {
	return &VkProvider{client: client}
}

func (p *VkProvider) GetPosts(source entity.Source) ([]entity.Post, error) {
	posts := make([]entity.Post, 0, 50)

	id, err := getGroupId(source.Link)
	if err != nil {
		return nil, err
	}

	groupId := "-" + id
	resp, err := p.client.WallGet(api.Params{
		"owner_id": groupId,
		"filter":   "all",
		"count":    getPostsAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("wall get unsuccessfull: %w : %s", VkError, err)
	}

posts:
	for _, item := range resp.Items {

		if len(item.Attachments) == 0 || item.IsDeleted {
			continue posts
		}

		tm := time.Unix(int64(item.Date), 0)

		post := entity.Post{
			Id:       item.ID,
			FromId:   item.FromID,
			Date:     tm,
			Likes:    item.Likes.Count,
			Reposts:  item.Reposts.Count,
			Comments: item.Comments.Count,
			Views:    item.Views.Count,
			Text:     item.Text,
		}

		for _, att := range item.Attachments {
			if t, ok := attachmentType[att.Type]; t != source.Category || !ok {
				continue posts
			}

			attachment := entity.Attachment{}
			switch att.Type {
			case "video":
				video, err := p.client.VideoGet(api.Params{
					"videos": strconv.Itoa(att.Video.OwnerID) + "_" + strconv.Itoa(att.Video.ID),
				})
				if err != nil {
					continue posts
				}

				if len(video.Items) == 0 {
					continue posts
				}

				attachment = entity.Attachment{
					Id:       att.Video.ID,
					Category: entity.Video,
					Link:     video.Items[0].Player,
					Duration: att.Video.Duration,
				}
			case "photo":
				var maxSize float64
				var imageURL string
				for _, size := range att.Photo.Sizes {
					if size.Height > maxSize {
						maxSize = size.Height
						imageURL = size.URL
					}
				}
				attachment = entity.Attachment{
					Id:       att.Photo.ID,
					Category: entity.Photo,
					Link:     imageURL,
					Duration: 0,
				}
			}

			post.Attachments = append(post.Attachments, attachment)

			time.Sleep(delay)
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (p *VkProvider) uploadPhoto(groupID, attachmentID int) (string, error) {
	path := filesPath + strconv.Itoa(attachmentID) + ".jpg"
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("photo was not opened: %w : %s", InternalError, err)
	}

	resp, err := p.client.UploadGroupWallPhoto(groupID, f)
	if err != nil {
		return "", fmt.Errorf("photo was not uploaded: %w : %s", VkError, err)
	}

	return resp[0].ToAttachment(), nil
}

func (p *VkProvider) uploadVideo(groupID, attachmentID int, videoName string) (string, error) {
	path := filesPath + strconv.Itoa(attachmentID) + ".mp4"
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("video was not opened: %w : %s", InternalError, err)
	}

	resp, err := p.client.UploadVideo(api.Params{
		"group_id": groupID,
		"name":     videoName,
	}, f)
	if err != nil {
		return "", fmt.Errorf("video was not uploaded: %w : %s", VkError, err)
	}

	return "video-" + strconv.Itoa(groupID) + "_" + strconv.Itoa(resp.VideoID), nil
}

func (p *VkProvider) PutPost(post entity.Post, groupName, groupTag, groupLink string, publishDate int64) error {
	var (
		attachments []string
		attString   string
		tagString   string
	)
	groupId, err := getGroupId(groupLink)
	if err != nil {
		return err
	}

	groupIdInt, _ := strconv.Atoi(groupId)

	defer deleteAttachments(post.Attachments)

	for _, att := range post.Attachments {
		switch att.Category {
		case entity.Video:
			err := downloadVideo(att)
			if err != nil {
				return err
			}
			attString, err = p.uploadVideo(groupIdInt, att.Id, groupName)
			if err != nil {
				return err
			}
		case entity.Photo:
			err := downloadPhoto(att)
			if err != nil {
				return err
			}
			attString, err = p.uploadPhoto(groupIdInt, att.Id)
			if err != nil {
				return err
			}
		}

		attachments = append(attachments, attString)
	}

	if groupTag != "" {
		tagString = "\n\n" + groupTag
	}

	_, err = p.client.WallPost(api.Params{
		"owner_id":     "-" + groupId,
		"attachments":  strings.Join(attachments, ","),
		"message":      post.Text + tagString,
		"publish_date": publishDate,
	})
	if err != nil {
		return fmt.Errorf("wall was not posted: %w : %s", VkError, err)
	}

	return nil
}

func getGroupId(link string) (string, error) {
	re := regexp.MustCompile("[0-9]+")
	id := re.FindAllString(link, -1)
	if len(id) == 0 {
		return "", fmt.Errorf("wrong link: %s", InternalError)
	}

	return id[0], nil
}

func downloadVideo(attachment entity.Attachment) error {
	c := make(chan error, 1)
	deadlineTime := time.Now().Add(MaxDownloadingTime)
	ctx, cancel := context.WithDeadline(context.TODO(), deadlineTime)
	defer cancel()

	go func() {
		result, err := goutubedl.New(ctx, attachment.Link, goutubedl.Options{})
		if err != nil {
			c <- err
			return
		}

		downloadResult, err := result.Download(ctx, "best")
		if err != nil {
			c <- err
			return
		}

		defer downloadResult.Close()

		filename := strconv.Itoa(attachment.Id)

		f, err := os.Create(filesPath + filename + ".ts")
		if err != nil {
			c <- err
		}

		defer f.Close()

		_, err = io.Copy(f, downloadResult)
		if err != nil {
			c <- err
			return
		}

		err = ffmpeg.Input(filesPath+filename+".ts").
			Output(filesPath+filename+".mp4", ffmpeg.KwArgs{"b:a": "128k"}).
			OverWriteOutput().
			WithTimeout(deadlineTime.Sub(time.Now())).
			Run()
		if err != nil {
			c <- err
			return
		}

		c <- nil
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("video downloading timeout %s: %w", attachment.Link, InternalError)
	case err := <-c:
		if err != nil {
			return fmt.Errorf("video was not downloaded: %w : %s", InternalError, err)
		}

		return err
	}
}

func downloadPhoto(attachment entity.Attachment) error {
	response, err := http.Get(attachment.Link)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("received non 200 response code: %w", InternalError)
	}

	filename := strconv.Itoa(attachment.Id)

	path := filesPath + filename + ".jpg"
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w : %s", InternalError, err)
	}
	defer file.Close()

	io.Copy(file, response.Body)

	return nil
}

func deleteAttachments(attachments []entity.Attachment) {
	files, _ := os.ReadDir(filesPath)

	for _, att := range attachments {
		for _, f := range files {
			filename := f.Name()
			if strings.Contains(filename, strconv.Itoa(att.Id)) {
				os.Remove(filesPath + filename)
			}
		}
	}
}
