package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Service provides a wrapper around the Google Drive API.
type Service struct {
	driveService *drive.Service
}

// NewService initializes a new Google Drive service wrapper using OAuth2 credentials and token.
func NewService(credentialsJSONPath string, tokenJSONPath string) (*Service, error) {
	ctx := context.Background()

	// 1. Read OAuth2 client credentials
	b, err := os.ReadFile(credentialsJSONPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client credentials file %s: %w", credentialsJSONPath, err)
	}

	// 2. Parse OAuth2 config (requesting full drive scope to upload/create/share)
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	// 3. Load or obtain token
	tok, err := tokenFromFile(tokenJSONPath)
	if err != nil {
		// Token not found or invalid, trigger local web server authentication flow
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("unable to authenticate via web flow: %w", err)
		}
		// Save token for future runs
		if err := saveToken(tokenJSONPath, tok); err != nil {
			return nil, fmt.Errorf("unable to save token to %s: %w", tokenJSONPath, err)
		}
	}

	// 4. Initialize Client and Drive Service
	client := config.Client(ctx, tok)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %w", err)
	}

	return &Service{driveService: srv}, nil
}

// UploadAndShareFile uploads a local file to a Google Drive folder and makes it publicly viewable.
// Returns the sharing webViewLink (URL).
func (s *Service) UploadAndShareFile(ctx context.Context, localFilePath string, fileName string, folderName string) (string, error) {
	// 1. Get or create the folder by name
	folderID, err := s.getOrCreateFolder(ctx, folderName)
	if err != nil {
		return "", fmt.Errorf("failed to get or create folder: %w", err)
	}

	existingFile, err := s.findFileByName(ctx, folderID, fileName)
	if err != nil {
		return "", fmt.Errorf("failed to search for existing file: %w", err)
	}
	if existingFile != nil {
		if err := s.ensureAnyoneReaderPermission(ctx, existingFile.Id); err != nil {
			return "", fmt.Errorf("failed to create sharing permission for existing file: %w", err)
		}
		return s.webViewLink(ctx, existingFile.Id)
	}

	// 2. Open the local file
	f, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open local file %s: %w", localFilePath, err)
	}
	defer f.Close()

	// 3. Define Google Drive metadata
	driveFile := &drive.File{
		Name:    fileName,
		Parents: []string{folderID},
	}

	// 4. Create (upload) file in Google Drive (supporting shared drives)
	uploadedFile, err := s.driveService.Files.Create(driveFile).Media(f).
		SupportsAllDrives(true).
		Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create file on Google Drive: %w", err)
	}

	// 5. Create permission (anyone with link can view, supporting shared drives)
	if err := s.ensureAnyoneReaderPermission(ctx, uploadedFile.Id); err != nil {
		return "", fmt.Errorf("failed to create sharing permission: %w", err)
	}

	// 6. Fetch file details to retrieve the sharing link (supporting shared drives)
	return s.webViewLink(ctx, uploadedFile.Id)
}

func (s *Service) findFileByName(ctx context.Context, folderID string, fileName string) (*drive.File, error) {
	escapedName := escapeDriveQueryString(fileName)
	escapedFolderID := escapeDriveQueryString(folderID)
	query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed = false", escapedName, escapedFolderID)
	listCall, err := s.driveService.Files.List().Q(query).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		PageSize(1).
		Fields("files(id,name,webViewLink)").
		Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if len(listCall.Files) == 0 {
		return nil, nil
	}
	return listCall.Files[0], nil
}

func (s *Service) ensureAnyoneReaderPermission(ctx context.Context, fileID string) error {
	permissions, err := s.driveService.Permissions.List(fileID).
		SupportsAllDrives(true).
		Fields("permissions(id,type,role)").
		Context(ctx).Do()
	if err != nil {
		return err
	}
	for _, permission := range permissions.Permissions {
		if permission.Type == "anyone" && (permission.Role == "reader" || permission.Role == "commenter" || permission.Role == "writer") {
			return nil
		}
	}

	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	_, err = s.driveService.Permissions.Create(fileID, permission).
		SupportsAllDrives(true).
		Context(ctx).Do()
	return err
}

func (s *Service) webViewLink(ctx context.Context, fileID string) (string, error) {
	fileDetails, err := s.driveService.Files.Get(fileID).
		SupportsAllDrives(true).
		Fields("webViewLink").
		Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to fetch sharing link from Google Drive: %w", err)
	}
	return fileDetails.WebViewLink, nil
}

func (s *Service) getOrCreateFolder(ctx context.Context, folderName string) (string, error) {
	// Search for folder by name (escaping single quotes for apostrophes in names like "Об'єкти", supporting shared drives)
	escapedName := escapeDriveQueryString(folderName)
	query := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and name = '%s' and trashed = false", escapedName)
	listCall, err := s.driveService.Files.List().Q(query).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("files(id, name)").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for folder '%s': %w", folderName, err)
	}

	if len(listCall.Files) > 0 {
		return listCall.Files[0].Id, nil
	}

	// Folder not found, create it (supporting shared drives)
	folderMetadata := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	newFolder, err := s.driveService.Files.Create(folderMetadata).
		SupportsAllDrives(true).
		Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create folder '%s': %w", folderName, err)
	}

	return newFolder.Id, nil
}

func escapeDriveQueryString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return replacer.Replace(value)
}

// getTokenFromWeb starts a local HTTP redirect server and opens the browser to authenticate.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	// 1. Start a local listener on a random free port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local port listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d", port)
	config.RedirectURL = redirectURL

	// 2. Create channels to receive auth code or errors
	codeChan := make(chan string)
	errChan := make(chan error)

	// 3. Define local HTTP handler to capture the authorization code
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "Authentication code not found in URL", http.StatusBadRequest)
				errChan <- fmt.Errorf("auth code query parameter missing")
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`
				<html>
				<head><title>Authorization Successful</title></head>
				<body style="font-family: Arial, sans-serif; text-align: center; margin-top: 60px; background-color: #f7f9fa; color: #333;">
					<div style="display: inline-block; padding: 30px; background: white; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
						<h2 style="color: #4CAF50; margin-bottom: 10px;">🔑 Авторизація успішна!</h2>
						<p style="font-size: 16px; margin-bottom: 20px;">Програма отримала доступ до Google Диску.</p>
						<p style="color: #666; font-size: 14px;">Тепер ви можете закрити цю вкладку та повернутися до програми.</p>
					</div>
				</body>
				</html>
			`))

			codeChan <- code
		}),
	}

	// 4. Run server in background
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 5. Open authorization URL in default browser
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Opening browser to authorize access: %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to automatically open browser (please open manually): %s: %w", authURL, err)
	}

	// 6. Wait for code or timeout
	var code string
	select {
	case code = <-codeChan:
		// Received authorization code
	case err := <-errChan:
		return nil, fmt.Errorf("error in local callback server: %w", err)
	case <-time.After(3 * time.Minute):
		_ = srv.Shutdown(context.Background())
		return nil, fmt.Errorf("authorization flow timed out after 3 minutes")
	}

	// 7. Shut down server
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShutdown)

	// 8. Exchange authorization code for token
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	return tok, nil
}

// tokenFromFile loads a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a local file.
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // Linux/BSD
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
