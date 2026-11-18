package main

import "github.com/lymuru/lymuru/backend"

// Cover and artist image downloads. The backend returns a response that
// describes any failure, so when one exists it is returned with a nil error:
// the frontend reads response.success / response.error instead of catching a
// rejected promise.

// DownloadCover saves a track or album cover next to the track's audio file.
func (a *App) DownloadCover(req backend.CoverDownloadRequest) (*backend.CoverDownloadResponse, error) {
	resp, err := backend.NewCoverClient().DownloadCover(req)
	return resolveMediaResponse("Cover", resp, err)
}

// DownloadHeader saves an artist's header image.
func (a *App) DownloadHeader(req backend.HeaderDownloadRequest) (*backend.HeaderDownloadResponse, error) {
	resp, err := backend.NewCoverClient().DownloadHeader(req)
	return resolveMediaResponse("Header", resp, err)
}

// DownloadAvatar saves an artist's avatar image.
func (a *App) DownloadAvatar(req backend.AvatarDownloadRequest) (*backend.AvatarDownloadResponse, error) {
	resp, err := backend.NewCoverClient().DownloadAvatar(req)
	return resolveMediaResponse("Avatar", resp, err)
}

// DownloadGalleryImage saves one image from an artist's gallery.
func (a *App) DownloadGalleryImage(req backend.GalleryImageDownloadRequest) (*backend.GalleryImageDownloadResponse, error) {
	resp, err := backend.NewCoverClient().DownloadGalleryImage(req)
	return resolveMediaResponse("Gallery", resp, err)
}

func resolveMediaResponse[T any](kind string, resp *T, err error) (*T, error) {
	if err != nil {
		backend.LogWarn("[%s] %v", kind, err)
	}
	if resp != nil {
		return resp, nil
	}
	return nil, err
}
