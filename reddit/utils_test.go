package reddit

import (
	"testing"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

func TestRedgifs(t *testing.T) {
	url1 := `https://redgifs.com/watch/rustyecstaticghostshrimp`

	if !redgifsRx.Match([]byte(url1)) {
		t.FailNow()
	}

	url2, err := getRedgifsURL(url1)
	if err != nil {
		t.Fatal(err)
	}

	if url2 == "" {
		t.FailNow()
	}
}

func TestGallery(t *testing.T) {
	url := `https://www.reddit.com/gallery/nz70m9`
	//url := `https://www.reddit.com/gallery/l3bf0z`
	client, err := reddit.NewReadonlyClient()
	if err != nil {
		t.Fatal(err)
	}

	if !galleryRx.Match([]byte(url)) {
		t.FailNow()
	}

	urls, err := getGalleryURLs(client, url)
	if err != nil {
		t.Fatal(err)
	}

	if len(urls) != 2 {
		t.FailNow()
	}

	for _, u := range urls {
		if u == "" {
			t.FailNow()
		}
	}
}

func TestImgur(t *testing.T) {
	url1 := `https://i.imgur.com/qBKC5AZ.gifv`
	if !imgurRx.Match([]byte(url1)) {
		t.FailNow()
	}

	url2, err := getImgurURL(url1)
	if err != nil {
		t.Fatal(err)
	}

	if url2 == "" {
		t.FailNow()
	}
}
