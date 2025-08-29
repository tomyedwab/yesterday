package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	yesterdaygo "github.com/tomyedwab/yesterday/clients/go"
)

func centerBox(contents tview.Primitive, width, height int) *tview.Flex {
	hflex := tview.NewFlex().SetFullScreen(false).SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(contents, width, 0, true).
		AddItem(tview.NewBox(), 0, 1, false)
	return tview.NewFlex().SetFullScreen(false).SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(hflex, height, 0, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

func createLoginPages(client *yesterdaygo.Client, pages *tview.Pages, mainPage *MainPage) {
	// Create "logging in" page
	var loggingInText = tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetTextAlign(tview.AlignCenter)
	loggingInText.SetBorder(true)
	pages.AddPage("LoggingIn", centerBox(loggingInText, 35, 3), true, true)

	// Create "logging in error" page
	var errorText = tview.NewTextView().
		SetTextColor(tcell.ColorRed).
		SetTextAlign(tview.AlignCenter)
	errorText.SetBorder(true)
	pages.AddPage("LoginError", errorText, true, true)

	// Create login form
	var loginUsername string
	var loginPassword string
	form := tview.NewForm().SetButtonsAlign(tview.AlignCenter)
	form.SetBorder(true)
	form.AddInputField("Username", "", 20, nil, func(username string) {
		loginUsername = username
	})
	form.AddInputField("Password", "", 20, nil, func(password string) {
		loginPassword = password
	})
	form.AddButton("Log in", func() {
		loggingInText.SetText(fmt.Sprintf("Logging in as %s...", loginUsername))
		_ = loginPassword
		pages.SwitchToPage("LoggingIn")
		if err := client.Login(context.Background(), loginUsername, loginPassword); err != nil {
			errorText.SetText(fmt.Sprintf("Error logging in: %s", err.Error()))
			pages.SwitchToPage("LoginError")
		} else {
			mainPage.update()
			pages.SwitchToPage("Main")
		}
	})

	pages.AddPage("Login", centerBox(form, 35, 9), true, true)
}

type UserData struct {
	ID       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
}

type UsersData struct {
	Users []UserData `json:"users"`
}

type MainPage struct {
	provider *yesterdaygo.DataProvider[UsersData]
	pages    *tview.Pages
}

func (m *MainPage) update() {
	users, err := m.provider.Get()
	if err != nil {
		var errorText = tview.NewTextView().
			SetTextColor(tcell.ColorRed).
			SetTextAlign(tview.AlignCenter).
			SetText(fmt.Sprintf("Error fetching users: %s", err.Error()))
		errorText.SetBorder(true)
		m.pages.AddPage("Main", errorText, true, true)
	}

	var usersList = tview.NewList().ShowSecondaryText(false)
	for index, user := range users.Users {
		usersList.AddItem(user.Username, fmt.Sprintf("ID %d", user.ID), rune(49+index), nil)
	}
	m.pages.AddPage("Main", usersList, true, true)
}

func main() {
	client := yesterdaygo.NewClient(
		"https://www.yesterday.localhost:8443",
		yesterdaygo.WithRefreshTokenPath(path.Join(os.Getenv("HOME"), ".yesterday", "token")),
	)
	err := client.RefreshAccessToken(context.Background())
	if err != nil {
		fmt.Println("Error refreshing access token:", err)
	}

	var app = tview.NewApplication()
	var pages = tview.NewPages()
	var mainPage = &MainPage{
		provider: yesterdaygo.NewDataProvider[UsersData](client, "/MBtskI6D/api/users", map[string]interface{}{}),
		pages:    pages,
	}

	pages.AddPage("Main", centerBox(tview.NewTextView().SetText("Welcome to Yesterday!"), 35, 9), true, true)
	createLoginPages(client, pages, mainPage)

	if client.IsAuthenticated() {
		mainPage.update()
		pages.SwitchToPage("Main")
	} else {
		pages.SwitchToPage("Login")
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 113 {
			app.Stop()
		}
		return event
	})
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
