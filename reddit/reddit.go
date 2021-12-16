package reddit

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Reddit struct {
	client *reddit.Client
	hashes map[string]bool
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
		hashes: make(map[string]bool),
	}, nil
}

func NewReadonly() (*Reddit, error) {
	client, err := reddit.NewReadonlyClient()
	if err != nil {
		return nil, err
	}

	return &Reddit{
		client: client,
		hashes: make(map[string]bool),
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
	if err := mkdirIfNotExist(dirname); err != nil {
		return err
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

		results, err := r.downloadURL(ctx, url)
		if err != nil {
			fmt.Println(err)
			continue
		}

		newDir := dirname
		if len(results) > 1 {
			newDir = fmt.Sprintf("%s/%s", dirname, filepath.Base(url))

			if err := mkdirIfNotExist(newDir); err != nil {
				return nil
			}
		}

		for _, result := range results {
			filepath := fmt.Sprintf("%s/%s", newDir, result.file)
			if err := r.saveFile(ctx, result.rc, filepath); err != nil {
				fmt.Println(err)
				continue
			}
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

type downloadResult struct {
	rc   io.ReadCloser
	file string
}

func (r *Reddit) downloadURL(ctx context.Context, URL string) ([]*downloadResult, error) {
	if redgifsRx.Match([]byte(URL)) {
		return downloadRedgifsURL(ctx, URL)
	}

	if galleryRx.Match([]byte(URL)) {
		return downloadGalleryURL(ctx, r.client, URL)
	}

	if imgurRx.Match([]byte(URL)) {
		return downloadImgurURL(ctx, URL)
	}

	result, err := downloadURL(ctx, URL)
	if err != nil {
		return nil, err
	}

	return []*downloadResult{result}, nil
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) {
	return rf(p)
}

var errDuplicatedFiles = errors.New("duplicated files")

func (r *Reddit) saveFile(ctx context.Context, rc io.ReadCloser, filepath string) error {
	defer rc.Close()

	_, err := os.Stat(filepath)
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer func() {
		out.Close()

		if errors.Is(err, errDuplicatedFiles) {
			os.Remove(filepath)
		}
	}()

	hasher := md5.New()
	reader := io.TeeReader(rc, hasher)

	_, err = io.Copy(out, readerFunc(func(b []byte) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		n, err := reader.Read(b)
		if err != nil {
			return 0, err
		}

		hash := hex.EncodeToString(hasher.Sum(nil))
		if r.hashes[hash] {
			return 0, errDuplicatedFiles
		}

		r.hashes[hash] = true
		return n, err
	}))

	return err
}
