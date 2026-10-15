package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/dotsaver/pkg/git"
	"github.com/user/dotsaver/pkg/manifest"
)

func TestSyncerSaveAndSync(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	localFile := filepath.Join(targetDir, "sample.conf")
	_ = os.WriteFile(localFile, []byte("setting = 1"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	_ = m.AddTarget("mytool", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(git.New())

	// Save
	saveRep, err := syncer.Save(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if len(saveRep.Saved) != 1 {
		t.Fatalf("Expected 1 saved item, got %d", len(saveRep.Saved))
	}

	// Change local file content to cause conflict
	_ = os.WriteFile(localFile, []byte("setting = 2"), 0644)

	conflicts, err := syncer.DetectConflicts(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	// Sync with autoBackup
	syncRep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(syncRep.BackedUp) != 1 {
		t.Fatalf("Expected 1 backed up file, got %d", len(syncRep.BackedUp))
	}

	// Check that backup file exists
	bakFile := localFile + ".bak"
	if _, err := os.Stat(bakFile); err != nil {
		t.Errorf("Backup file %s not found: %v", bakFile, err)
	}

	// Check that localFile was restored to repo content ("setting = 1")
	data, _ := os.ReadFile(localFile)
	if string(data) != "setting = 1" {
		t.Errorf("Expected restored content 'setting = 1', got '%s'", string(data))
	}
}

func TestSyncerSaveMissingSourceFiles(t *testing.T) {
	// Review Focus 1: Missing source files on Save must record a non-fatal warning
	// and continue saving the remaining targets.
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	missingFile := filepath.Join(targetDir, "missing.conf")
	existingFile := filepath.Join(targetDir, "existing.conf")
	_ = os.WriteFile(existingFile, []byte("existing content"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("prog1")
	_ = m.AddTarget("prog1", manifest.Target{
		OS:   "windows",
		Path: missingFile,
	})
	_ = m.AddTarget("prog1", manifest.Target{
		OS:   "windows",
		Path: existingFile,
	})

	syncer := NewSyncer(nil)
	if syncer.Git == nil {
		t.Fatalf("Expected NewSyncer(nil) to initialize default GitService")
	}

	rep, err := syncer.Save(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("Expected non-fatal save error, got: %v", err)
	}

	if len(rep.Warnings) != 1 {
		t.Fatalf("Expected 1 warning for missing file, got %d: %v", len(rep.Warnings), rep.Warnings)
	}
	if !strings.Contains(rep.Warnings[0], "path does not exist") {
		t.Errorf("Expected warning to mention 'path does not exist', got: %s", rep.Warnings[0])
	}

	if len(rep.Saved) != 1 {
		t.Fatalf("Expected 1 saved item, got %d: %v", len(rep.Saved), rep.Saved)
	}
	if !strings.Contains(rep.Saved[0], "existing.conf") {
		t.Errorf("Expected saved item to contain 'existing.conf', got: %s", rep.Saved[0])
	}

	// Verify existing file was indeed copied to repo cache
	dstRepoPath := syncer.RepoTargetDir(repoDir, "prog1", "windows", "existing.conf")
	if _, err := os.Stat(dstRepoPath); err != nil {
		t.Fatalf("Expected repo cache file to exist: %s, err: %v", dstRepoPath, err)
	}
}

func TestSyncerBackupCollisionTimestampFallback(t *testing.T) {
	// Review Focus 2: Existing .bak conflict collision - if <file>.bak exists,
	// create timestamped <file>.bak.YYYYMMDDHHMMSS to avoid overwriting existing backup.
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	localFile := filepath.Join(targetDir, "app.cfg")
	bakFile := localFile + ".bak"

	// Original repo content
	_ = os.WriteFile(localFile, []byte("repo-v1"), 0644)
	m := manifest.New()
	_, _ = m.AddProgram("app")
	_ = m.AddTarget("app", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(git.New())
	if _, err := syncer.Save(repoDir, m, "windows"); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Existing .bak already on disk from previous backup
	originalBakContent := []byte("old-backup-content")
	_ = os.WriteFile(bakFile, originalBakContent, 0644)

	// Local file is modified to repo-v2
	_ = os.WriteFile(localFile, []byte("local-v2"), 0644)

	// Sync with autoBackup=true
	syncRep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(syncRep.BackedUp) != 1 {
		t.Fatalf("Expected 1 backed up file, got %d", len(syncRep.BackedUp))
	}

	timestampBakPath := syncRep.BackedUp[0]
	if !strings.HasPrefix(timestampBakPath, localFile+".bak.") {
		t.Fatalf("Expected timestamped backup path starting with '%s.bak.', got: %s", localFile, timestampBakPath)
	}

	// Verify original .bak was NOT overwritten
	existingBakData, err := os.ReadFile(bakFile)
	if err != nil {
		t.Fatalf("Failed to read original .bak file: %v", err)
	}
	if string(existingBakData) != string(originalBakContent) {
		t.Errorf("Original .bak was overwritten! Expected '%s', got '%s'", originalBakContent, existingBakData)
	}

	// Verify new timestamped backup contains the local-v2 content
	newBakData, err := os.ReadFile(timestampBakPath)
	if err != nil {
		t.Fatalf("Failed to read timestamped .bak file: %v", err)
	}
	if string(newBakData) != "local-v2" {
		t.Errorf("Expected new backup to contain 'local-v2', got '%s'", string(newBakData))
	}

	// Verify local file was restored to repo-v1
	localData, _ := os.ReadFile(localFile)
	if string(localData) != "repo-v1" {
		t.Errorf("Expected local file restored to 'repo-v1', got '%s'", string(localData))
	}
}

func TestSyncerBackupCollisionMultipleTimestamp(t *testing.T) {
	// Test collision when both .bak and timestamped .bak already exist
	tempDir := t.TempDir()
	localFile := filepath.Join(tempDir, "test.txt")
	bakFile := localFile + ".bak"
	_ = os.WriteFile(localFile, []byte("new-data"), 0644)
	_ = os.WriteFile(bakFile, []byte("bak-1"), 0644)

	timestamp := time.Now().Format("20060102150405")
	existingTimestampBak := localFile + ".bak." + timestamp
	_ = os.WriteFile(existingTimestampBak, []byte("bak-timestamp"), 0644)

	createdBak, err := createBackup(localFile)
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	expectedPrefix := existingTimestampBak + "_"
	if !strings.HasPrefix(createdBak, expectedPrefix) {
		t.Errorf("Expected duplicate timestamp fallback with prefix '%s', got '%s'", expectedPrefix, createdBak)
	}

	data, err := os.ReadFile(createdBak)
	if err != nil {
		t.Fatalf("Failed to read created backup: %v", err)
	}
	if string(data) != "new-data" {
		t.Errorf("Expected 'new-data', got '%s'", string(data))
	}
}

func TestSyncerSyncWithoutAutoBackup(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	localFile := filepath.Join(targetDir, "sample.conf")
	_ = os.WriteFile(localFile, []byte("v1"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	_ = m.AddTarget("mytool", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(git.New())
	_, _ = syncer.Save(repoDir, m, "windows")

	// Modify local file
	_ = os.WriteFile(localFile, []byte("v2"), 0644)

	// Sync with autoBackup=false
	syncRep, err := syncer.Sync(repoDir, m, "windows", false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(syncRep.Conflicts) != 1 {
		t.Fatalf("Expected 1 conflict item, got %d", len(syncRep.Conflicts))
	}
	if len(syncRep.BackedUp) != 0 {
		t.Errorf("Expected 0 backed up files when autoBackup=false, got %d", len(syncRep.BackedUp))
	}
	if len(syncRep.Updated) != 0 {
		t.Errorf("Expected 0 updated files when autoBackup=false and conflicting, got %d", len(syncRep.Updated))
	}

	// Local file should remain unmodified
	localData, _ := os.ReadFile(localFile)
	if string(localData) != "v2" {
		t.Errorf("Expected local file unchanged 'v2', got '%s'", string(localData))
	}
}

func TestSyncerSyncLocalFileMissing(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	localFile := filepath.Join(targetDir, "sample.conf")
	_ = os.WriteFile(localFile, []byte("from-repo"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	_ = m.AddTarget("mytool", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(git.New())
	_, _ = syncer.Save(repoDir, m, "windows")

	// Remove local file
	_ = os.Remove(localFile)

	// DetectConflicts should report 0 conflicts since local file doesn't exist
	conflicts, err := syncer.DetectConflicts(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("Expected 0 conflicts for missing local file, got %d", len(conflicts))
	}

	// Sync should restore file without creating a backup
	syncRep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(syncRep.BackedUp) != 0 {
		t.Errorf("Expected 0 backups for previously non-existent file, got %d", len(syncRep.BackedUp))
	}
	if len(syncRep.Updated) != 1 {
		t.Errorf("Expected 1 updated file, got %d", len(syncRep.Updated))
	}

	data, _ := os.ReadFile(localFile)
	if string(data) != "from-repo" {
		t.Errorf("Expected restored content 'from-repo', got '%s'", string(data))
	}
}

func TestSyncerSyncAlreadyInSync(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target")
	_ = os.MkdirAll(targetDir, 0755)

	localFile := filepath.Join(targetDir, "sample.conf")
	_ = os.WriteFile(localFile, []byte("identical"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	_ = m.AddTarget("mytool", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(git.New())
	_, _ = syncer.Save(repoDir, m, "windows")

	// Sync without changes
	syncRep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(syncRep.Updated) != 0 {
		t.Errorf("Expected 0 updated files when already in sync, got %d", len(syncRep.Updated))
	}
	if len(syncRep.BackedUp) != 0 {
		t.Errorf("Expected 0 backups when already in sync, got %d", len(syncRep.BackedUp))
	}
}

func TestSyncerDirectorySaveAndSync(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetDir := filepath.Join(tempDir, "target_dir")
	_ = os.MkdirAll(filepath.Join(targetDir, "subdir"), 0755)

	file1 := filepath.Join(targetDir, "config.json")
	file2 := filepath.Join(targetDir, "subdir", "sub.txt")
	_ = os.WriteFile(file1, []byte(`{"key": 1}`), 0644)
	_ = os.WriteFile(file2, []byte("nested"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("appdir")
	_ = m.AddTarget("appdir", manifest.Target{
		OS:   "windows",
		Path: targetDir,
	})

	syncer := NewSyncer(git.New())

	// Save directory
	saveRep, err := syncer.Save(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("Save directory failed: %v", err)
	}
	if len(saveRep.Saved) != 1 {
		t.Fatalf("Expected 1 saved item, got %d", len(saveRep.Saved))
	}

	// Modify one file inside directory
	_ = os.WriteFile(file2, []byte("modified nested"), 0644)

	conflicts, err := syncer.DetectConflicts(repoDir, m, "windows")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict for directory, got %d", len(conflicts))
	}

	// Sync with autoBackup
	syncRep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(syncRep.BackedUp) != 1 {
		t.Fatalf("Expected 1 backed up directory, got %d", len(syncRep.BackedUp))
	}

	// Verify backup directory contains modified file
	backedUpFile2 := filepath.Join(syncRep.BackedUp[0], "subdir", "sub.txt")
	data, err := os.ReadFile(backedUpFile2)
	if err != nil {
		t.Fatalf("Failed to read backed up nested file: %v", err)
	}
	if string(data) != "modified nested" {
		t.Errorf("Expected 'modified nested' in backup, got '%s'", string(data))
	}

	// Verify targetDir file2 is restored to original
	dataRestored, _ := os.ReadFile(file2)
	if string(dataRestored) != "nested" {
		t.Errorf("Expected restored nested file 'nested', got '%s'", string(dataRestored))
	}
}

func TestSyncerNilManifest(t *testing.T) {
	syncer := NewSyncer(nil)
	if _, err := syncer.Save("repo", nil, "windows"); err == nil {
		t.Errorf("Expected error saving nil manifest, got nil")
	}
	if _, err := syncer.Sync("repo", nil, "windows", true); err == nil {
		t.Errorf("Expected error syncing nil manifest, got nil")
	}
	conflicts, err := syncer.DetectConflicts("repo", nil, "windows")
	if err != nil || len(conflicts) != 0 {
		t.Errorf("Expected empty conflicts for nil manifest, err=%v, len=%d", err, len(conflicts))
	}
}

func TestSyncerSyncMissingRepoEntry(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	targetFile := filepath.Join(tempDir, "local.conf")
	_ = os.WriteFile(targetFile, []byte("data"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("unregistered")
	_ = m.AddTarget("unregistered", manifest.Target{
		OS:   "windows",
		Path: targetFile,
	})

	syncer := NewSyncer(nil)
	// Sync without prior Save -> repo cache has no entry
	rep, err := syncer.Sync(repoDir, m, "windows", true)
	if err != nil {
		t.Fatalf("Sync returned unexpected error: %v", err)
	}
	if len(rep.Warnings) != 1 {
		t.Fatalf("Expected 1 warning for missing repo cache entry, got %d", len(rep.Warnings))
	}
	if !strings.Contains(rep.Warnings[0], "not found in repository cache") {
		t.Errorf("Expected warning 'not found in repository cache', got: %s", rep.Warnings[0])
	}

	// Verify local file was NOT backed up or deleted
	if _, err := os.Stat(targetFile + ".bak"); err == nil {
		t.Errorf("Local file was prematurely backed up when repoEntry was missing!")
	}
	if _, err := os.Stat(targetFile); err != nil {
		t.Errorf("Local file was removed or lost when repoEntry was missing!")
	}
}

func TestSyncerResolveAliasBackslashes(t *testing.T) {
	alias1 := resolveAlias(manifest.Target{Path: `C:\Users\tester\AppData\Roaming\app\settings.json`}, "app")
	if alias1 != "settings.json" {
		t.Errorf("Expected 'settings.json', got '%s'", alias1)
	}

	alias2 := resolveAlias(manifest.Target{Path: `/home/user/.config/app/settings.json`}, "app")
	if alias2 != "settings.json" {
		t.Errorf("Expected 'settings.json', got '%s'", alias2)
	}

	aliasExplicit := resolveAlias(manifest.Target{Path: `/path/to/file`, Alias: "custom_alias"}, "app")
	if aliasExplicit != "custom_alias" {
		t.Errorf("Expected 'custom_alias', got '%s'", aliasExplicit)
	}
}

func TestSyncerCurrentOSCaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	localFile := filepath.Join(tempDir, "test.conf")
	_ = os.WriteFile(localFile, []byte("data"), 0644)

	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	_ = m.AddTarget("mytool", manifest.Target{
		OS:   "windows",
		Path: localFile,
	})

	syncer := NewSyncer(nil)
	rep, err := syncer.Save(repoDir, m, " WiNdOwS ")
	if err != nil || len(rep.Saved) != 1 {
		t.Fatalf("Save with mixed case OS failed: %v, saved: %v", err, rep.Saved)
	}

	conflicts, err := syncer.DetectConflicts(repoDir, m, "WINDOWS")
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("DetectConflicts with uppercase OS failed: %v, conflicts: %v", err, conflicts)
	}

	syncRep, err := syncer.Sync(repoDir, m, "Windows", true)
	if err != nil {
		t.Fatalf("Sync with Title case OS failed: %v", err)
	}
	if len(syncRep.Warnings) != 0 {
		t.Errorf("Unexpected warnings: %v", syncRep.Warnings)
	}
}

