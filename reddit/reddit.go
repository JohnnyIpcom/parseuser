package reddit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Reddit struct {
	client *reddit.Client
}

func New(clientID string, clientSecret string, username string, password string) (*Reddit, error) {
	credentials := reddit.Credentials{
		ID:       clientID,
		Secret:   clientSecret,
		Username: username,
		Password: password,
	}

	client, err := reddit.NewClient(credentials)
	if err != nil {
		return nil, err
	}

	return &Reddit{
		client: client,
	}, nil
}

func NewReadonly() (*Reddit, error) {
	client, err := reddit.NewReadonlyClient()
	if err != nil {
		return nil, err
	}

	return &Reddit{
		client: client,
	}, nil
}

type postsProvider func(ctx context.Context, lastID string) ([]*reddit.Post, *reddit.Response, error)

func (r *Reddit) getPostsFromUser(ctx context.Context, username string) postsProvider {
	return func(ctx context.Context, lastID string) ([]*reddit.Post, *reddit.Response, error) {
		return r.client.User.PostsOf(
			ctx,
			username,
			&reddit.ListUserOverviewOptions{
				ListOptions: reddit.ListOptions{
					After: lastID,
				},
				Sort: "new",
				Time: "all",
			},
		)
	}
}

func (r *Reddit) getPostsFromSubreddit(ctx context.Context, subreddit string) postsProvider {
	return func(ctx context.Context, lastID string) ([]*reddit.Post, *reddit.Response, error) {
		return r.client.Subreddit.NewPosts(
			ctx,
			subreddit,
			&reddit.ListOptions{
				After: lastID,
			},
		)
	}
}

func (r *Reddit) getURLs(ctx context.Context, p postsProvider) chan string {
	urls := make(chan string)
	go func() {
		defer close(urls)

		lastID := ""
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			posts, _, err := p(ctx, lastID)
			if err != nil {
				return
			}

			if len(posts) <= 0 {
				return
			}

			lastID = posts[len(posts)-1].FullID

			for _, post := range posts {
				urls <- post.URL
			}
		}
	}()

	return urls
}

func (r *Reddit) GetURLsFromUser(ctx context.Context, username string) chan string {
	return r.getURLs(ctx, r.getPostsFromUser(ctx, username))
}

func (r *Reddit) GetURLsFromSubreddit(ctx context.Context, subreddit string) chan string {
	return r.getURLs(ctx, r.getPostsFromSubreddit(ctx, subreddit))
}

func (r *Reddit) Download(ctx context.Context, dirname string, urls chan string) error {
	if _, err := os.Stat(dirname); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(dirname, 0666); err != nil {
			return err
		}
	}

	uniqueURLs := make(map[string]bool)
	for url := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if uniqueURLs[url] {
			continue
		}

		err := r.downloadURL(ctx, dirname, url)
		if err != nil {
			fmt.Println(err)
			continue
		}

		uniqueURLs[url] = true
	}

	return nil
}

var (
	redgifsRx = regexp.MustCompile(`^https?://[wm.]*redgifs.com/watch/([a-zA-Z0-9_-]+).*$`)
	galleryRx = regexp.MustCompile(`^https?://[a-zA-Z0-9.]{0,4}reddit.com/gallery/([a-zA-Z0-9]+).*$`)
	imgurRx   = regexp.MustCompile(`^https?://i.imgur.com/([a-zA-Z0-9]+).*$`)
)

func (r *Reddit) downloadURL(ctx context.Context, dirpath string, URL string) error {
	if redgifsRx.Match([]byte(URL)) {
		return downloadRedgifsURL(ctx, dirpath, URL)
	}

	if galleryRx.Match([]byte(URL)) {
		return downloadGalleryURL(ctx, r.client, dirpath, URL)
	}

	if imgurRx.Match([]byte(URL)) {
		return downloadImgurURL(ctx, dirpath, URL)
	}

	return downloadURL(ctx, dirpath, URL)
}
