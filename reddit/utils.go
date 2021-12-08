package reddit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/anaskhan96/soup"
	"github.com/vartanbeno/go-reddit/v2/reddit"
	"golang.org/x/net/html/charset"
)

func getRedgifsURL(URL string) (string, error) {
	resp, err := soup.Get(URL)
	if err != nil {
		return "", err
	}

	html := soup.HTMLParse(resp)
	links := html.FindAll("")
	for _, link := range links {
		if !strings.Contains(link.HTML(), "content") {
			continue
		}

		attrs := link.Attrs()
		if content, ok := attrs["content"]; ok {
			if strings.Contains(content, "redgifs.com") && strings.Contains(content, "mp4") && strings.Contains(content, "-mobile") {
				return content, nil
			}
		}
	}

	return "", fmt.Errorf("no appropriate video in redgifs link '%s'", URL)
}

func getGalleryURLs(client *reddit.Client, URL string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(context.Background(), req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	utf8Body, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(utf8Body)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0)

	html := soup.HTMLParse(string(bytes))
	links := html.FindAll("a")
	for _, link := range links {
		if !strings.Contains(link.HTML(), "href") {
			continue
		}

		attrs := link.Attrs()
		if href, ok := attrs["href"]; ok {
			if strings.Contains(href, "preview.redd.it") {
				url, err := convertFromPreviewURL(href)
				if err != nil {
					continue
				}

				res = append(res, url)
			}
		}
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no appropriate content in gallery link '%s'", URL)
	}

	return res, nil
}

func getImgurURL(URL string) (string, error) {
	resp, err := soup.Get(URL)
	if err != nil {
		return "", err
	}

	html := soup.HTMLParse(resp)
	links := html.FindAll("")
	for _, link := range links {
		if !strings.Contains(link.HTML(), "content") {
			continue
		}

		attrs := link.Attrs()
		if content, ok := attrs["content"]; ok {
			if strings.Contains(content, "i.imgur.com") && strings.Contains(content, "mp4") {
				return content, nil
			}
		}
	}

	return "", fmt.Errorf("no appropriate video in imgur link '%s'", URL)
}

func convertFromPreviewURL(URL string) (string, error) {
	previewURL, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	iRedditURL, err := url.Parse("https://i.redd.it/")
	if err != nil {
		return "", err
	}

	res, err := iRedditURL.Parse(previewURL.Path)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

func downloadRedgifsURL(ctx context.Context, dirpath string, URL string) error {
	url, err := getRedgifsURL(URL)
	if err != nil {
		return err
	}

	return downloadURL(ctx, dirpath, url)
}

func downloadGalleryURL(ctx context.Context, client *reddit.Client, dirpath string, URL string) error {
	urls, err := getGalleryURLs(client, URL)
	if err != nil {
		return err
	}

	newDirpath := fmt.Sprintf("%s/%s", dirpath, filepath.Base(URL))

	if _, err := os.Stat(newDirpath); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(newDirpath, 0666); err != nil {
			return err
		}
	}

	for _, url := range urls {
		err := downloadURL(ctx, newDirpath, url)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadImgurURL(ctx context.Context, dirpath string, URL string) error {
	base := filepath.Base(URL)
	if strings.HasSuffix(base, ".jpg") || strings.HasSuffix(base, ".png") {
		return downloadURL(ctx, dirpath, URL)
	}

	url, err := getImgurURL(URL)
	if err != nil {
		return err
	}

	return downloadURL(ctx, dirpath, url)
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) {
	return rf(p)
}

func downloadURL(ctx context.Context, dirpath string, URL string) error {
	fmt.Printf("Downloading '%s'...\n", URL)

	resp, err := http.Get(URL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got wrong status %s", resp.Status)
	}

	filepath := fmt.Sprintf("%s/%s", dirpath, filepath.Base(URL))

	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(filepath)
		if err != nil {
			return err
		}

		defer out.Close()

		_, err = io.Copy(out, readerFunc(func(b []byte) (int, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				return resp.Body.Read(b)
			}
		}))

		if err != nil {
			return err
		}
	}

	return nil
}
