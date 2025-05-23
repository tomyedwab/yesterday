package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/users/auth"
	"github.com/tomyedwab/yesterday/users/state"
)

const userServiceURL = "http://localhost:8081" // Assuming user service runs locally

func runRequest(url string, method string, body []byte, accessToken string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send request, status code: %d", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func doLogin() (string, error) {
	fmt.Println("Please log in.")
	// Read in username and password from stdin
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')

	reqBody := map[string]string{
		"username": strings.TrimSpace(username),
		"password": strings.TrimSpace(password),
	}
	reqJson, _ := json.Marshal(reqBody)
	resp, err := http.Post(userServiceURL+"/api/login?app=0001-0001", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to login, status code: %d", resp.StatusCode)
	}

	// Read the refresh token from the response cookie
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "YRT" {
			os.WriteFile("refresh_token.txt", []byte(cookie.Value), 0644)
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("no refresh token found")
}

func getAccessToken() (string, error) {
	refreshTokenBytes, err := os.ReadFile("refresh_token.txt")
	var refreshToken string
	if err == nil {
		refreshToken = string(refreshTokenBytes)
	} else {
		refreshToken, err = doLogin()
		if err != nil {
			return "", err
		}
	}

	var resp *http.Response
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest("POST", userServiceURL+"/api/refresh", nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Cookie", "YRT="+refreshToken)
		resp, err = http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		resp.Body.Close()

		// Try logging in and refreshing again
		if i < 1 {
			var loginErr error
			refreshToken, loginErr = doLogin()
			if loginErr != nil {
				return "", loginErr
			}
		}
	}

	defer resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to refresh token, status %d", resp.StatusCode)
	}

	// Read the refresh token from the response cookie
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "YRT" {
			os.WriteFile("refresh_token.txt", []byte(cookie.Value), 0644)
		}
	}

	var refreshResponse auth.RefreshResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read refresh response body: %w", err)
	}
	err = json.Unmarshal(body, &refreshResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal refresh response: %w", err)
	}

	return refreshResponse.AccessToken, nil
}

func listApplications(accessToken string) error {
	url := userServiceURL + "/api/listapplications"

	body, err := runRequest(url, "GET", nil, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Applications:")
	var apps []state.Application
	err = json.Unmarshal(body, &apps)
	if err != nil {
		return err
	}
	for _, app := range apps {
		fmt.Printf("- %s (ID: %s, Host: %s)\n", app.DisplayName, app.ID, app.HostName)
	}
	return nil
}

func listUsers(accessToken string) error {
	url := userServiceURL + "/api/listusers"

	body, err := runRequest(url, "GET", nil, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Users:")
	var users []struct {
		ID       int
		Username string
	}
	err = json.Unmarshal(body, &users)
	if err != nil {
		return err
	}
	for _, user := range users {
		fmt.Printf("- %s [%d]\n", user.Username, user.ID)
	}
	return nil
}

func getUserProfile(accessToken, userId, application string) error {
	url := userServiceURL + "/api/getuserprofile?userId=" + userId + "&application=" + application

	body, err := runRequest(url, "GET", nil, accessToken)
	if err != nil {
		return err
	}

	var profile struct {
		UserId      int
		Application string
		Profile     map[string]interface{}
	}
	err = json.Unmarshal(body, &profile)
	if err != nil {
		return err
	}

	fmt.Printf("Profile for user %d in application %s:\n", profile.UserId, profile.Application)
	for k, v := range profile.Profile {
		fmt.Printf("- %s: %v\n", k, v)
	}
	return nil
}

func createUser(accessToken, username string) error {
	url := userServiceURL + "/api/publish?cid=" + uuid.New().String()

	event := state.UserAddedEvent{
		GenericEvent: events.GenericEvent{
			Id:   0,
			Type: "users:ADD_USER",
		},
		Username: username,
	}
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("User created successfully")
	return nil
}

func setUserProfile(accessToken string, userId int, applicationId, profile string) error {
	url := userServiceURL + "/api/publish?cid=" + uuid.New().String()

	event := state.UserProfileUpdatedEvent{
		GenericEvent: events.GenericEvent{
			Id:   0,
			Type: "users:USER_PROFILE_UPDATED",
		},
		UserID:        userId,
		ApplicationID: applicationId,
		ProfileData:   profile,
	}
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("User profile updated successfully")
	return nil
}

type ChangePasswordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func changePassword(accessToken, username, password string) error {
	reqBody := ChangePasswordRequest{
		Username: username,
		Password: password,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := userServiceURL + "/api/changepw"

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Password changed successfully for user:", username)
	return nil
}

func addApplication(accessToken, id, displayName, hostName string) error {
	url := userServiceURL + "/api/publish?cid=" + uuid.New().String()

	event := state.ApplicationAddedEvent{
		GenericEvent: events.GenericEvent{
			Id:   0, // Set by the server
			Type: "users:ADD_APPLICATION",
		},
		ApplicationID: id,
		DisplayName:   displayName,
		HostName:      hostName,
	}
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Application added successfully")
	return nil
}

func deleteApplication(accessToken, id string) error {
	url := userServiceURL + "/api/publish?cid=" + uuid.New().String()

	event := state.ApplicationDeletedEvent{
		GenericEvent: events.GenericEvent{
			Id:   0, // Set by the server
			Type: "users:DELETE_APPLICATION",
		},
		ApplicationID: id,
	}
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Application deleted successfully")
	return nil
}

func updateApplicationHostName(accessToken, id, hostName string) error {
	url := userServiceURL + "/api/publish?cid=" + uuid.New().String()

	event := state.ApplicationHostNameUpdatedEvent{
		GenericEvent: events.GenericEvent{
			Id:   0, // Set by the server
			Type: "users:UPDATE_APPLICATION_HOSTNAME",
		},
		ApplicationID: id,
		HostName:      hostName,
	}
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = runRequest(url, "POST", jsonData, accessToken)
	if err != nil {
		return err
	}

	fmt.Println("Application hostname updated successfully")
	return nil
}

func main() {
	// Basic command routing
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <command> [options]", os.Args[0])
	}

	accessToken, err := getAccessToken()
	if err != nil {
		log.Fatalf("Error getting access token: %v", err)
	}

	command := os.Args[1]

	switch command {
	case "listusers":
		err := listUsers(accessToken)
		if err != nil {
			log.Fatalf("Error listing users: %v", err)
		}

	case "listapplications":
		err := listApplications(accessToken)
		if err != nil {
			log.Fatalf("Error listing applications: %v", err)
		}

	case "getuserprofile":
		userProfileCmd := flag.NewFlagSet("getuserprofile", flag.ExitOnError)
		userId := userProfileCmd.String("user", "", "User ID to get profile for")
		application := userProfileCmd.String("application", "", "Application name")

		userProfileCmd.Parse(os.Args[2:])
		if *userId == "" || *application == "" {
			log.Fatalf("Usage: %s getuserprofile --user <user_id> --application <application_name>", os.Args[0])
		}

		err := getUserProfile(accessToken, *userId, *application)
		if err != nil {
			log.Fatalf("Error getting user profile: %v", err)
		}

	case "createuser":
		createUserCmd := flag.NewFlagSet("createuser", flag.ExitOnError)
		username := createUserCmd.String("username", "", "Username to create")

		createUserCmd.Parse(os.Args[2:])

		if *username == "" {
			log.Fatalf("Usage: %s createuser --username <username>", os.Args[0])
		}

		err := createUser(accessToken, *username)
		if err != nil {
			log.Fatalf("Error creating user: %v", err)
		}

	case "addapplication":
		addAppCmd := flag.NewFlagSet("addapplication", flag.ExitOnError)
		appId := addAppCmd.String("id", "", "Application ID")
		displayName := addAppCmd.String("name", "", "Application display name")
		hostName := addAppCmd.String("host", "", "Application host name")

		addAppCmd.Parse(os.Args[2:])

		if *appId == "" || *displayName == "" {
			log.Fatalf("Usage: %s addapplication --id <id> --name <display_name> [--host <host_name>]", os.Args[0])
		}

		err := addApplication(accessToken, *appId, *displayName, *hostName)
		if err != nil {
			log.Fatalf("Error adding application: %v", err)
		}

	case "deleteapplication":
		deleteAppCmd := flag.NewFlagSet("deleteapplication", flag.ExitOnError)
		appId := deleteAppCmd.String("id", "", "Application ID to delete")

		deleteAppCmd.Parse(os.Args[2:])

		if *appId == "" {
			log.Fatalf("Usage: %s deleteapplication --id <id>", os.Args[0])
		}

		err := deleteApplication(accessToken, *appId)
		if err != nil {
			log.Fatalf("Error deleting application: %v", err)
		}

	case "updateapplicationhostname":
		updateHostCmd := flag.NewFlagSet("updateapplicationhostname", flag.ExitOnError)
		appId := updateHostCmd.String("id", "", "Application ID to update")
		hostName := updateHostCmd.String("host", "", "New host name for the application")

		updateHostCmd.Parse(os.Args[2:])

		if *appId == "" || *hostName == "" {
			log.Fatalf("Usage: %s updateapplicationhostname --id <id> --host <host_name>", os.Args[0])
		}

		err := updateApplicationHostName(accessToken, *appId, *hostName)
		if err != nil {
			log.Fatalf("Error updating application hostname: %v", err)
		}

	case "setuserprofile":
		setUserProfileCmd := flag.NewFlagSet("setuserprofile", flag.ExitOnError)
		userId := setUserProfileCmd.Int("user", 0, "User ID to set profile for")
		applicationID := setUserProfileCmd.String("application", "", "Application name")
		profile := setUserProfileCmd.String("profile", "", "Profile data")

		setUserProfileCmd.Parse(os.Args[2:])

		if *userId == 0 || *applicationID == "" || *profile == "" {
			log.Fatalf("Usage: %s setuserprofile --user <user_id> --application <application_id> --profile <profile_data>", os.Args[0])
		}

		err := setUserProfile(accessToken, *userId, *applicationID, *profile)
		if err != nil {
			log.Fatalf("Error setting user profile: %v", err)
		}

	case "changepw":
		changePwCmd := flag.NewFlagSet("changepw", flag.ExitOnError)
		username := changePwCmd.String("username", "", "Username to change password for")
		password := changePwCmd.String("password", "", "New password")

		changePwCmd.Parse(os.Args[2:])

		if *username == "" || *password == "" {
			log.Fatalf("Usage: %s changepw --username <username> --password <password>", os.Args[0])
		}

		err := changePassword(accessToken, *username, *password)
		if err != nil {
			log.Fatalf("Error changing password: %v", err)
		}

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
