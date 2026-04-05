package ui

import (
	"fmt"
	"strconv"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "myssh/internal/app"
	"myssh/internal/domain"
)

type screen struct {
	service *appsvc.Service
	dataDir string

	window   fyne.Window
	profiles []domain.Profile
	selected int

	list *widget.List

	nameEntry     *widget.Entry
	usernameEntry *widget.Entry
	hostEntry     *widget.Entry
	portEntry     *widget.Entry
	authSelect    *widget.Select
	secretEntry   *widget.Entry
	statusLabel   *widget.Label
	secretHint    *widget.Label
}

func Run(service *appsvc.Service, dataDir string) error {
	a := app.NewWithID("ro.mihai.myssh")
	w := a.NewWindow("MySSH")
	w.Resize(fyne.NewSize(1080, 720))

	s := &screen{
		service:  service,
		dataDir:  dataDir,
		window:   w,
		selected: -1,
	}

	if err := s.reloadProfiles(); err != nil {
		return err
	}

	s.build()
	w.ShowAndRun()
	return nil
}

func (s *screen) build() {
	title := widget.NewLabel("MySSH")
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := widget.NewLabel("Fast, lightweight SSH profiles")
	info := widget.NewLabel("Secrets are not stored in plain text. Keyring integration is the next step.")
	info.Wrapping = fyne.TextWrapWord

	newButton := widget.NewButton("New SSH", func() {
		s.selected = -1
		s.clearForm()
		s.list.UnselectAll()
		s.statusLabel.SetText("Creating a new SSH profile.")
	})

	s.list = widget.NewList(
		func() int { return len(s.profiles) },
		func() fyne.CanvasObject {
			return widget.NewLabel("profile")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			profile := s.profiles[id]
			label.SetText(fmt.Sprintf("%s  |  %s@%s:%d", profile.Name, profile.Username, profile.Host, profile.Port))
		},
	)
	s.list.OnSelected = func(id widget.ListItemID) {
		s.selected = id
		s.loadProfile(s.profiles[id])
	}

	sidebar := container.NewBorder(
		container.NewVBox(title, subtitle, info),
		newButton,
		nil,
		nil,
		s.list,
	)

	s.nameEntry = widget.NewEntry()
	s.nameEntry.SetPlaceHolder("Production API")

	s.usernameEntry = widget.NewEntry()
	s.usernameEntry.SetPlaceHolder("root")

	s.hostEntry = widget.NewEntry()
	s.hostEntry.SetPlaceHolder("server.example.com or 10.0.0.12")

	s.portEntry = widget.NewEntry()
	s.portEntry.SetPlaceHolder("22")
	s.portEntry.SetText(strconv.Itoa(domain.DefaultSSHPort))

	s.secretHint = widget.NewLabel("")
	s.secretHint.Wrapping = fyne.TextWrapWord

	s.authSelect = widget.NewSelect([]string{
		string(domain.AuthPassword),
		string(domain.AuthPrivateKey),
		string(domain.AuthAgent),
	}, func(value string) {
		s.updateSecretHint(value)
	})
	s.authSelect.SetSelected(string(domain.AuthAgent))

	s.secretEntry = widget.NewPasswordEntry()
	s.secretEntry.SetPlaceHolder("Temporary input only")
	s.secretEntry.Disable()

	s.updateSecretHint(string(domain.AuthAgent))

	s.statusLabel = widget.NewLabel("Ready.")
	s.statusLabel.Wrapping = fyne.TextWrapWord

	saveButton := widget.NewButton("Save Profile", func() {
		s.saveProfile()
	})

	form := widget.NewForm(
		widget.NewFormItem("Name", s.nameEntry),
		widget.NewFormItem("Username", s.usernameEntry),
		widget.NewFormItem("Domain / IP", s.hostEntry),
		widget.NewFormItem("Port", s.portEntry),
		widget.NewFormItem("Auth", s.authSelect),
		widget.NewFormItem("Password / Key", s.secretEntry),
	)

	right := container.NewVBox(
		widget.NewLabel("Connection Details"),
		form,
		s.secretHint,
		saveButton,
		widget.NewSeparator(),
		widget.NewLabel("Storage"),
		widget.NewLabel(s.dataDir),
		widget.NewSeparator(),
		s.statusLabel,
	)

	s.window.SetContent(container.NewHSplit(sidebar, container.NewPadded(right)))
}

func (s *screen) reloadProfiles() error {
	profiles, err := s.service.ListProfiles()
	if err != nil {
		return err
	}
	s.profiles = profiles
	return nil
}

func (s *screen) loadProfile(profile domain.Profile) {
	s.nameEntry.SetText(profile.Name)
	s.usernameEntry.SetText(profile.Username)
	s.hostEntry.SetText(profile.Host)
	s.portEntry.SetText(strconv.Itoa(profile.Port))
	s.authSelect.SetSelected(string(profile.AuthKind))
	s.secretEntry.SetText("")
	s.statusLabel.SetText("Loaded profile metadata. Secret values stay outside plain-text storage.")
}

func (s *screen) clearForm() {
	s.nameEntry.SetText("")
	s.usernameEntry.SetText("")
	s.hostEntry.SetText("")
	s.portEntry.SetText(strconv.Itoa(domain.DefaultSSHPort))
	s.authSelect.SetSelected(string(domain.AuthAgent))
	s.secretEntry.SetText("")
}

func (s *screen) currentProfile() (domain.Profile, error) {
	port, err := strconv.Atoi(s.portEntry.Text)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("port must be a valid number")
	}

	profile := domain.Profile{
		Name:     s.nameEntry.Text,
		Username: s.usernameEntry.Text,
		Host:     s.hostEntry.Text,
		Port:     port,
		AuthKind: domain.AuthKind(s.authSelect.Selected),
	}

	if s.selected >= 0 && s.selected < len(s.profiles) {
		profile.ID = s.profiles[s.selected].ID
		profile.SecretRef = s.profiles[s.selected].SecretRef
	}

	return profile, nil
}

func (s *screen) saveProfile() {
	profile, err := s.currentProfile()
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	if _, err := s.service.AddProfile(profile); err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	if err := s.reloadProfiles(); err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	s.list.Refresh()
	s.statusLabel.SetText("Profile saved. Secrets still require dedicated secure storage integration.")
}

func (s *screen) updateSecretHint(auth string) {
	if s.secretHint == nil {
		return
	}

	switch auth {
	case string(domain.AuthPassword):
		s.secretHint.SetText("Password storage is intentionally disabled for now. Next step: OS keyring integration.")
	case string(domain.AuthPrivateKey):
		s.secretHint.SetText("Private key storage is intentionally disabled for now. Next step: secure keyring or file reference flow.")
	default:
		s.secretHint.SetText("Agent mode is the safest default for the MVP and avoids local secret persistence.")
	}
}
