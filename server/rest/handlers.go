package rest

import (
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/marcopeocchi/yt-dlp-web-ui/server/config"
	"github.com/marcopeocchi/yt-dlp-web-ui/server/utils"
)

type DirectoryEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	SHASum      string    `json:"shaSum"`
	ModTime     time.Time `json:"modTime"`
	IsVideo     bool      `json:"isVideo"`
	IsDirectory bool      `json:"isDirectory"`
}

func walkDir(root string) (*[]DirectoryEntry, error) {
	files := []DirectoryEntry{}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, d := range dirs {
		if !utils.IsValidEntry(d) {
			continue
		}

		path := filepath.Join(root, d.Name())

		info, err := d.Info()
		if err != nil {
			return nil, err
		}

		files = append(files, DirectoryEntry{
			Path:        path,
			Name:        d.Name(),
			Size:        info.Size(),
			SHASum:      utils.ShaSumString(path),
			IsVideo:     utils.IsVideo(d),
			IsDirectory: d.IsDir(),
			ModTime:     info.ModTime(),
		})
	}

	return &files, err
}

type ListRequest struct {
	SubDir  string `json:"subdir"`
	OrderBy string `json:"orderBy"`
}

func ListDownloaded(ctx *fiber.Ctx) error {
	root := config.Instance().GetConfig().DownloadPath
	req := new(ListRequest)

	err := ctx.BodyParser(req)
	if err != nil {
		return err
	}

	files, err := walkDir(filepath.Join(root, req.SubDir))
	if err != nil {
		return err
	}

	if req.OrderBy == "modtime" {
		sort.SliceStable(*files, func(i, j int) bool {
			return (*files)[i].ModTime.After((*files)[j].ModTime)
		})
	}

	ctx.Status(http.StatusOK)
	return ctx.JSON(files)
}

type DeleteRequest = DirectoryEntry

func DeleteFile(ctx *fiber.Ctx) error {
	req := new(DeleteRequest)

	err := ctx.BodyParser(req)
	if err != nil {
		return err
	}

	sum := utils.ShaSumString(req.Path)
	if sum != req.SHASum {
		return errors.New("shasum mismatch")
	}

	err = os.Remove(req.Path)
	if err != nil {
		return err
	}

	ctx.Status(fiber.StatusOK)
	return ctx.JSON("ok")
}

func SendFile(ctx *fiber.Ctx) error {
	path := ctx.Params("id")

	if path == "" {
		return errors.New("inexistent path")
	}

	decoded, err := hex.DecodeString(path)
	if err != nil {
		return err
	}
	decodedStr := string(decoded)

	root := config.Instance().GetConfig().DownloadPath

	// TODO: further path / file validations
	if strings.Contains(filepath.Dir(decodedStr), root) {
		// ctx.Response().Header.Set(
		// 	"Content-Disposition",
		// 	"inline; filename="+filepath.Base(decodedStr),
		// )
		ctx.SendStatus(fiber.StatusOK)
		return ctx.SendFile(decodedStr)
	}

	return ctx.SendStatus(fiber.StatusUnauthorized)
}