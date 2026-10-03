// Package fetch downloads cover images.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/voilelab/epub_toolbox/internal/i18n"
)

// MaxSize caps a downloaded image.
const MaxSize = 20 << 20

// Image downloads url. In the browser (js/wasm) the request goes through
// fetch() and fails unless the host allows CORS.
func Image(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if runtime.GOOS == "js" && !errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf(i18n.T("%w（該網站可能不允許跨來源讀取，請下載圖片後改用上傳）",
				"%w (the site may not allow cross-origin reads; download the image and upload it instead)"), err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxSize {
		return nil, fmt.Errorf(i18n.T("圖片超過 %d MiB", "image exceeds %d MiB"), MaxSize>>20)
	}
	return b, nil
}
