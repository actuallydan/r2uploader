package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Profile struct {
	Name        string
	Credentials CloudflareCredentials
}

type ProfileManager struct {
	ProfilesPath string
	Profiles     []Profile
}

func NewProfileManager() (*ProfileManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get home directory: %v", err)
	}

	// Create .r2uploader directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".r2uploader")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create config directory: %v", err)
	}

	profilesPath := filepath.Join(configDir, "profiles.json")
	pm := &ProfileManager{
		ProfilesPath: profilesPath,
	}

	// Load existing profiles
	if err := pm.loadProfiles(); err != nil {
		// If file doesn't exist, that's fine - start with empty profiles
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return pm, nil
}

func (pm *ProfileManager) loadProfiles() error {
	data, err := os.ReadFile(pm.ProfilesPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &pm.Profiles)
}

func (pm *ProfileManager) saveProfiles() error {
	data, err := json.MarshalIndent(pm.Profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal profiles: %v", err)
	}

	return os.WriteFile(pm.ProfilesPath, data, 0600)
}

func (pm *ProfileManager) addProfile(name string, creds CloudflareCredentials) error {
	// Check if profile name already exists
	for _, p := range pm.Profiles {
		if p.Name == name {
			return fmt.Errorf("profile '%s' already exists", name)
		}
	}

	pm.Profiles = append(pm.Profiles, Profile{
		Name:        name,
		Credentials: creds,
	})

	return pm.saveProfiles()
}

func (pm *ProfileManager) getCredentials() (CloudflareCredentials, error) {
	if len(pm.Profiles) == 0 {
		return pm.handleNoProfiles()
	}

	fmt.Println("\nAvailable options:")
	// List profiles first
	for i, p := range pm.Profiles {
		fmt.Printf("%d. Use profile: %s\n", i+1, p.Name)
	}
	// Then list the new options
	fmt.Printf("%d. New upload (don't save)\n", len(pm.Profiles)+1)
	fmt.Printf("%d. Create new profile\n", len(pm.Profiles)+2)

	choice := getInput("\nSelect an option (1-" + fmt.Sprint(len(pm.Profiles)+2) + "): ")

	// Parse the choice
	idx := 0
	_, err := fmt.Sscanf(choice, "%d", &idx)
	if err != nil || idx < 1 || idx > len(pm.Profiles)+2 {
		return CloudflareCredentials{}, fmt.Errorf("invalid selection")
	}

	// Check if it's one of the profiles
	if idx <= len(pm.Profiles) {
		return pm.Profiles[idx-1].Credentials, nil
	}

	// Handle special options
	if idx == len(pm.Profiles)+1 {
		return getCredentialsFromUser()
	}
	return pm.createNewProfile()
}

func (pm *ProfileManager) handleNoProfiles() (CloudflareCredentials, error) {
	fmt.Println("\nNo saved profiles found.")
	fmt.Println("1. New upload (don't save)")
	fmt.Println("2. Create new profile")

	choice := getInput("\nSelect an option (1-2): ")
	switch choice {
	case "1":
		return getCredentialsFromUser()
	case "2":
		return pm.createNewProfile()
	default:
		return CloudflareCredentials{}, fmt.Errorf("invalid selection")
	}
}

func (pm *ProfileManager) createNewProfile() (CloudflareCredentials, error) {
	name := ""
	for name == "" {
		name = getInput("Enter profile name: ")
	}

	creds, err := getCredentialsFromUser()
	if err != nil {
		return CloudflareCredentials{}, err
	}

	err = pm.addProfile(name, creds)
	if err != nil {
		return CloudflareCredentials{}, err
	}

	fmt.Printf("Profile '%s' saved successfully!\n", name)
	return creds, nil
}
