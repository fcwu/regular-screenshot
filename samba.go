package main

import (
	"context"
	"io"
	"net"
	"path/filepath"
	"time"

	"github.com/hirochachacha/go-smb2"
)

type (
	Samba interface {
		Start() error
		Stop() error
		WithCredentials(username, password string) Samba
		UploadScreenshot(input io.Reader) error
	}

	_Samba struct {
		session *smb2.Session
		mount   *smb2.Share

		Host     string
		Share    string
		Path     string
		Username string
		Password string
	}
)

func NewSamba(host, share, path string) Samba {
	return &_Samba{Host: host, Share: share, Path: path}
}

func (s *_Samba) Start() error {
	var (
		d   = &smb2.Dialer{}
		err error
	)
	if s.Username != "" || s.Password != "" {
		d = &smb2.Dialer{
			Initiator: &smb2.NTLMInitiator{
				User:     s.Username,
				Password: s.Password,
			},
		}
	}
	conn, err := net.Dial("tcp", s.Host)
	if err != nil {
		return err
	}

	s.session, err = d.DialContext(context.TODO(), conn)
	if err != nil {
		return err
	}

	s.mount, err = s.session.Mount(s.Share)
	if err != nil {
		return err
	}

	if s.Path != "" {
		if err := s.mount.MkdirAll(filepath.Join(s.Host, s.Path), 0755); err != nil {
			return err
		}
	}

	return nil
}

func (s *_Samba) Stop() error {
	// nil pointer violation
	// _ = s.mount.Umount()
	// return s.session.Logoff()
	return nil
}

func (s *_Samba) WithCredentials(username, password string) Samba {
	s.Username = username
	s.Password = password
	return s
}

// UploadScreenshot will upload the screenshot to the samba share with the given path
// The filename will be the current timestamp in the format of "2006-01-02T15:04:05.png"
func (s *_Samba) UploadScreenshot(input io.Reader) error {
	filename := time.Now().Format("2006-01-02T150405.png")
	dstFile, err := s.mount.Create(filepath.Join(s.Path, filename))
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, input)
	if err != nil {
		return err
	}
	return nil
}
