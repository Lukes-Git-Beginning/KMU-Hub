package project

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	projects    map[uuid.UUID]*models.Project
	members     map[uuid.UUID]map[uuid.UUID]*models.ProjectMember // projectID -> userID -> member
	keys        map[string]bool
	statuses    map[uuid.UUID][]models.ProjectStatus // projectID -> statuses
	preferences map[string]*models.UserProjectPreference // "userID:projectID" -> pref

	createErr    error
	getErr       error
	updateErr    error
	archiveErr   error
	addMemberErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		projects:    make(map[uuid.UUID]*models.Project),
		members:     make(map[uuid.UUID]map[uuid.UUID]*models.ProjectMember),
		keys:        make(map[string]bool),
		statuses:    make(map[uuid.UUID][]models.ProjectStatus),
		preferences: make(map[string]*models.UserProjectPreference),
	}
}

func (m *MockRepository) Create(ctx context.Context, project *models.Project) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.projects[project.ID] = project
	m.keys[project.ProjectKey] = true
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	p, ok := m.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MockRepository) List(ctx context.Context, userID uuid.UUID, isAdmin bool, includeArchived bool) ([]models.ProjectWithDetails, error) {
	var result []models.ProjectWithDetails
	for _, p := range m.projects {
		if !includeArchived && p.ArchivedAt != nil {
			continue
		}
		if !isAdmin {
			if _, ok := m.members[p.ID]; !ok {
				continue
			}
			if _, ok := m.members[p.ID][userID]; !ok {
				continue
			}
		}
		memberCount := 0
		if mems, ok := m.members[p.ID]; ok {
			memberCount = len(mems)
		}
		result = append(result, models.ProjectWithDetails{
			Project:     *p,
			MemberCount: memberCount,
		})
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, project *models.Project) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.projects[project.ID] = project
	return nil
}

func (m *MockRepository) Archive(ctx context.Context, projectID uuid.UUID) error {
	if m.archiveErr != nil {
		return m.archiveErr
	}
	p, ok := m.projects[projectID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now()
	p.ArchivedAt = &now
	return nil
}

func (m *MockRepository) GetProjectKey(ctx context.Context, projectID uuid.UUID) (string, error) {
	p, ok := m.projects[projectID]
	if !ok {
		return "", ErrNotFound
	}
	return p.ProjectKey, nil
}

func (m *MockRepository) KeyExists(ctx context.Context, key string) (bool, error) {
	return m.keys[key], nil
}

func (m *MockRepository) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	if m.addMemberErr != nil {
		return m.addMemberErr
	}
	if _, ok := m.members[projectID]; !ok {
		m.members[projectID] = make(map[uuid.UUID]*models.ProjectMember)
	}
	if _, exists := m.members[projectID][userID]; exists {
		return nil // ON CONFLICT DO NOTHING
	}
	m.members[projectID][userID] = &models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		AddedAt:   time.Now(),
	}
	return nil
}

func (m *MockRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	if mems, ok := m.members[projectID]; ok {
		delete(mems, userID)
	}
	return nil
}

func (m *MockRepository) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	if mems, ok := m.members[projectID]; ok {
		if mem, exists := mems[userID]; exists {
			mem.Role = role
		}
	}
	return nil
}

func (m *MockRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	if mems, ok := m.members[projectID]; ok {
		if mem, exists := mems[userID]; exists {
			return mem, nil
		}
	}
	return nil, ErrNotProjectMember
}

func (m *MockRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	var result []models.ProjectMember
	if mems, ok := m.members[projectID]; ok {
		for _, mem := range mems {
			result = append(result, *mem)
		}
	}
	return result, nil
}

func (m *MockRepository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	if mems, ok := m.members[projectID]; ok {
		_, exists := mems[userID]
		return exists, nil
	}
	return false, nil
}

func (m *MockRepository) CountOwners(ctx context.Context, projectID uuid.UUID) (int, error) {
	count := 0
	if mems, ok := m.members[projectID]; ok {
		for _, mem := range mems {
			if mem.Role == models.ProjectRoleOwner {
				count++
			}
		}
	}
	return count, nil
}

func (m *MockRepository) SaveAsTemplate(ctx context.Context, sourceID uuid.UUID, newName string, newKey string, createdBy uuid.UUID) (*models.Project, error) {
	source, ok := m.projects[sourceID]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now()
	tmpl := &models.Project{
		ID:               uuid.New(),
		Name:             newName,
		Description:      source.Description,
		ProjectKey:       newKey,
		NextTaskNumber:   1,
		IsTemplate:       true,
		TemplateSourceID: &sourceID,
		CreatedBy:        createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	m.projects[tmpl.ID] = tmpl
	m.keys[newKey] = true
	return tmpl, nil
}

func (m *MockRepository) GetForTemplate(ctx context.Context, projectID uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	p, ok := m.projects[projectID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	return p, m.statuses[projectID], nil
}

func (m *MockRepository) GetUserPreference(ctx context.Context, userID, projectID uuid.UUID) (*models.UserProjectPreference, error) {
	key := userID.String() + ":" + projectID.String()
	pref, ok := m.preferences[key]
	if !ok {
		return nil, nil
	}
	return pref, nil
}

func (m *MockRepository) SetUserPreference(ctx context.Context, pref *models.UserProjectPreference) error {
	key := pref.UserID.String() + ":" + pref.ProjectID.String()
	m.preferences[key] = pref
	return nil
}

// Test helper to add a member directly
func (m *MockRepository) addMemberDirect(projectID, userID uuid.UUID, role string) {
	if _, ok := m.members[projectID]; !ok {
		m.members[projectID] = make(map[uuid.UUID]*models.ProjectMember)
	}
	m.members[projectID][userID] = &models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		AddedAt:   time.Now(),
	}
}

// Test helper to add a project directly
func (m *MockRepository) addProject(p *models.Project) {
	m.projects[p.ID] = p
	m.keys[p.ProjectKey] = true
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "My Project",
		ProjectKey: "MP",
		CreatedBy:  uuid.New(),
	}

	project, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, project.ID)
	assert.Equal(t, "My Project", project.Name)
	assert.Equal(t, "MP", project.ProjectKey)
	assert.Equal(t, 1, project.NextTaskNumber)
	assert.False(t, project.IsTemplate)
	assert.Nil(t, project.ArchivedAt)

	// Verify creator was added as owner
	members, _ := repo.ListMembers(context.Background(), project.ID)
	require.Len(t, members, 1)
	assert.Equal(t, input.CreatedBy, members[0].UserID)
	assert.Equal(t, models.ProjectRoleOwner, members[0].Role)
}

func TestService_Create_WithDescription(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	desc := "A test project description"
	input := CreateInput{
		Name:        "Test Project",
		Description: &desc,
		ProjectKey:  "TP",
		CreatedBy:   uuid.New(),
	}

	project, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, project.Description)
	assert.Equal(t, "A test project description", *project.Description)
}

func TestService_Create_KeyNormalizesToUppercase(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "Test",
		ProjectKey: "mp",
		CreatedBy:  uuid.New(),
	}

	project, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "MP", project.ProjectKey)
}

func TestService_Create_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "",
		ProjectKey: "MP",
		CreatedBy:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_NameTooShort(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "AB",
		ProjectKey: "MP",
		CreatedBy:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrNameTooShort)
}

func TestService_Create_EmptyKey(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "Project",
		ProjectKey: "",
		CreatedBy:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrKeyRequired)
}

func TestService_Create_KeyTooShort(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "Project",
		ProjectKey: "M",
		CreatedBy:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrKeyTooShort)
}

func TestService_Create_KeyTooLong(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "Project",
		ProjectKey: "ABCDEFGHIJK", // 11 chars
		CreatedBy:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrKeyTooLong)
}

func TestService_Create_KeyInvalidChars(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tests := []struct {
		name string
		key  string
	}{
		{"special chars", "A-B"},
		{"spaces", "A B"},
		{"underscore", "A_B"},
		{"dot", "A.B"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), CreateInput{
				Name:       "Project",
				ProjectKey: tc.key,
				CreatedBy:  uuid.New(),
			})
			assert.ErrorIs(t, err, ErrKeyInvalidChars)
		})
	}
}

func TestService_Create_DuplicateKey(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Create first project
	_, err := svc.Create(context.Background(), CreateInput{
		Name:       "Project 1",
		ProjectKey: "MP",
		CreatedBy:  uuid.New(),
	})
	require.NoError(t, err)

	// Try duplicate key
	_, err = svc.Create(context.Background(), CreateInput{
		Name:       "Project 2",
		ProjectKey: "MP",
		CreatedBy:  uuid.New(),
	})
	assert.ErrorIs(t, err, ErrKeyAlreadyExists)
}

// ============================================================================
// Get Tests
// ============================================================================

func TestService_Get_AsMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	userID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Test",
		ProjectKey: "TST",
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, userID, models.ProjectRoleOwner)

	project, err := svc.Get(context.Background(), projectID, userID, false)

	require.NoError(t, err)
	assert.Equal(t, projectID, project.ID)
}

func TestService_Get_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Test",
		ProjectKey: "TST",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Admin is NOT a member but can still access
	adminID := uuid.New()
	project, err := svc.Get(context.Background(), projectID, adminID, true)

	require.NoError(t, err)
	assert.Equal(t, projectID, project.ID)
}

func TestService_Get_NotMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Test",
		ProjectKey: "TST",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.Get(context.Background(), projectID, uuid.New(), false)

	assert.ErrorIs(t, err, ErrNotProjectMember)
}

func TestService_Get_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New(), false)

	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_AsMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	otherID := uuid.New()

	// Project user is member of
	p1 := &models.Project{ID: uuid.New(), Name: "Mine", ProjectKey: "MN", CreatedBy: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.addProject(p1)
	repo.addMemberDirect(p1.ID, userID, models.ProjectRoleOwner)

	// Project user is NOT member of
	p2 := &models.Project{ID: uuid.New(), Name: "Other", ProjectKey: "OT", CreatedBy: otherID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.addProject(p2)
	repo.addMemberDirect(p2.ID, otherID, models.ProjectRoleOwner)

	projects, err := svc.List(context.Background(), userID, false, false)

	require.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, "Mine", projects[0].Name)
}

func TestService_List_AsAdmin_SeesAll(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	adminID := uuid.New()

	p1 := &models.Project{ID: uuid.New(), Name: "P1", ProjectKey: "P1", CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	p2 := &models.Project{ID: uuid.New(), Name: "P2", ProjectKey: "P2", CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.addProject(p1)
	repo.addProject(p2)

	projects, err := svc.List(context.Background(), adminID, true, false)

	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Old Name",
		ProjectKey: "ON",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	newName := "New Name"
	project, err := svc.Update(context.Background(), projectID, ownerID, false, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", project.Name)
}

func TestService_Update_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	adminID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Old",
		ProjectKey: "OL",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	newName := "New"
	project, err := svc.Update(context.Background(), projectID, adminID, true, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New", project.Name)
}

func TestService_Update_AsMember_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	memberID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)

	newName := "Changed"
	_, err := svc.Update(context.Background(), projectID, memberID, false, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrNotProjectOwner)
}

func TestService_Update_Archived_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	archivedAt := time.Now()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Archived",
		ProjectKey: "AR",
		CreatedBy:  ownerID,
		ArchivedAt: &archivedAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	newName := "Changed"
	_, err := svc.Update(context.Background(), projectID, ownerID, false, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrProjectArchived)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	emptyName := ""
	_, err := svc.Update(context.Background(), projectID, ownerID, false, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

// ============================================================================
// Archive Tests
// ============================================================================

func TestService_Archive_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	err := svc.Archive(context.Background(), projectID, ownerID, false)

	require.NoError(t, err)
	assert.NotNil(t, repo.projects[projectID].ArchivedAt)
}

func TestService_Archive_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	err := svc.Archive(context.Background(), projectID, uuid.New(), true)

	require.NoError(t, err)
}

func TestService_Archive_AsMember_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	memberID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)

	err := svc.Archive(context.Background(), projectID, memberID, false)

	assert.ErrorIs(t, err, ErrNotProjectOwner)
}

func TestService_Archive_AlreadyArchived_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	archivedAt := time.Now()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		ArchivedAt: &archivedAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	err := svc.Archive(context.Background(), projectID, ownerID, false)

	assert.ErrorIs(t, err, ErrProjectArchived)
}

// ============================================================================
// AddMember Tests
// ============================================================================

func TestService_AddMember_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	newUserID := uuid.New()
	err := svc.AddMember(context.Background(), projectID, newUserID, ownerID, false, models.ProjectRoleMember)

	require.NoError(t, err)
	assert.NotNil(t, repo.members[projectID][newUserID])
	assert.Equal(t, models.ProjectRoleMember, repo.members[projectID][newUserID].Role)
}

func TestService_AddMember_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	newUserID := uuid.New()
	err := svc.AddMember(context.Background(), projectID, newUserID, uuid.New(), true, models.ProjectRoleViewer)

	require.NoError(t, err)
	assert.Equal(t, models.ProjectRoleViewer, repo.members[projectID][newUserID].Role)
}

func TestService_AddMember_AsMember_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	memberID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)

	err := svc.AddMember(context.Background(), projectID, uuid.New(), memberID, false, models.ProjectRoleMember)

	assert.ErrorIs(t, err, ErrNotProjectOwner)
}

func TestService_AddMember_InvalidRole(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.AddMember(context.Background(), uuid.New(), uuid.New(), uuid.New(), true, "superadmin")

	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestService_AddMember_ArchivedProject_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	archivedAt := time.Now()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Archived",
		ProjectKey: "AR",
		CreatedBy:  uuid.New(),
		ArchivedAt: &archivedAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	err := svc.AddMember(context.Background(), projectID, uuid.New(), uuid.New(), true, models.ProjectRoleMember)

	assert.ErrorIs(t, err, ErrProjectArchived)
}

// ============================================================================
// RemoveMember Tests
// ============================================================================

func TestService_RemoveMember_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	memberID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)

	err := svc.RemoveMember(context.Background(), projectID, memberID, ownerID, false)

	require.NoError(t, err)
	_, exists := repo.members[projectID][memberID]
	assert.False(t, exists)
}

func TestService_RemoveMember_CannotRemoveLastOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	// Admin tries to remove the only owner
	err := svc.RemoveMember(context.Background(), projectID, ownerID, uuid.New(), true)

	assert.ErrorIs(t, err, ErrCannotRemoveOnlyOwner)
}

func TestService_RemoveMember_CanRemoveOneOfMultipleOwners(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	owner1ID := uuid.New()
	owner2ID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  owner1ID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, owner1ID, models.ProjectRoleOwner)
	repo.addMemberDirect(projectID, owner2ID, models.ProjectRoleOwner)

	err := svc.RemoveMember(context.Background(), projectID, owner2ID, owner1ID, false)

	require.NoError(t, err)
}

func TestService_RemoveMember_AsMember_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	memberID := uuid.New()
	otherID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)
	repo.addMemberDirect(projectID, otherID, models.ProjectRoleMember)

	err := svc.RemoveMember(context.Background(), projectID, otherID, memberID, false)

	assert.ErrorIs(t, err, ErrNotProjectOwner)
}

// ============================================================================
// UpdateMemberRole Tests
// ============================================================================

func TestService_UpdateMemberRole_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	memberID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)
	repo.addMemberDirect(projectID, memberID, models.ProjectRoleMember)

	err := svc.UpdateMemberRole(context.Background(), projectID, memberID, ownerID, false, models.ProjectRoleViewer)

	require.NoError(t, err)
	assert.Equal(t, models.ProjectRoleViewer, repo.members[projectID][memberID].Role)
}

func TestService_UpdateMemberRole_CannotDemoteLastOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, ownerID, models.ProjectRoleOwner)

	// Admin tries to demote the last owner
	err := svc.UpdateMemberRole(context.Background(), projectID, ownerID, uuid.New(), true, models.ProjectRoleMember)

	assert.ErrorIs(t, err, ErrCannotRemoveOnlyOwner)
}

func TestService_UpdateMemberRole_InvalidRole(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.UpdateMemberRole(context.Background(), uuid.New(), uuid.New(), uuid.New(), true, "invalid")

	assert.ErrorIs(t, err, ErrInvalidRole)
}

// ============================================================================
// ListMembers Tests
// ============================================================================

func TestService_ListMembers_AsMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	repo.addMemberDirect(projectID, userID, models.ProjectRoleOwner)

	members, err := svc.ListMembers(context.Background(), projectID, userID, false)

	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestService_ListMembers_NotMember_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	repo.addProject(&models.Project{
		ID:         projectID,
		Name:       "Project",
		ProjectKey: "PR",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.ListMembers(context.Background(), projectID, uuid.New(), false)

	assert.ErrorIs(t, err, ErrNotProjectMember)
}

// ============================================================================
// SaveAsTemplate Tests
// ============================================================================

func TestService_SaveAsTemplate_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	sourceID := uuid.New()
	repo.addProject(&models.Project{
		ID:         sourceID,
		Name:       "Source Project",
		ProjectKey: "SP",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	tmpl, err := svc.SaveAsTemplate(context.Background(), sourceID, "My Template", "TMPL", uuid.New())

	require.NoError(t, err)
	assert.True(t, tmpl.IsTemplate)
	assert.Equal(t, "My Template", tmpl.Name)
	assert.Equal(t, "TMPL", tmpl.ProjectKey)
	assert.Equal(t, &sourceID, tmpl.TemplateSourceID)
}

func TestService_SaveAsTemplate_SourceNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.SaveAsTemplate(context.Background(), uuid.New(), "Template", "TM", uuid.New())

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_SaveAsTemplate_DuplicateKey(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	sourceID := uuid.New()
	repo.addProject(&models.Project{
		ID:         sourceID,
		Name:       "Source",
		ProjectKey: "SP",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.SaveAsTemplate(context.Background(), sourceID, "Template", "SP", uuid.New())

	assert.ErrorIs(t, err, ErrKeyAlreadyExists)
}

func TestService_SaveAsTemplate_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	sourceID := uuid.New()
	repo.addProject(&models.Project{
		ID:         sourceID,
		Name:       "Source",
		ProjectKey: "SP",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.SaveAsTemplate(context.Background(), sourceID, "", "TM", uuid.New())

	assert.ErrorIs(t, err, ErrNameRequired)
}

// ============================================================================
// CreateFromTemplate Tests
// ============================================================================

func TestService_CreateFromTemplate_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	templateID := uuid.New()
	desc := "Template description"
	repo.addProject(&models.Project{
		ID:          templateID,
		Name:        "Template",
		Description: &desc,
		ProjectKey:  "TM",
		IsTemplate:  true,
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	repo.statuses[templateID] = []models.ProjectStatus{
		{ID: uuid.New(), ProjectID: templateID, Name: "To Do", SortOrder: 0, IsDefault: true},
		{ID: uuid.New(), ProjectID: templateID, Name: "Done", SortOrder: 1, IsClosed: true},
	}

	creatorID := uuid.New()
	project, err := svc.CreateFromTemplate(context.Background(), templateID, "New Project", "NP", creatorID)

	require.NoError(t, err)
	assert.Equal(t, "New Project", project.Name)
	assert.Equal(t, "NP", project.ProjectKey)
	assert.False(t, project.IsTemplate)
	assert.Equal(t, &templateID, project.TemplateSourceID)
	require.NotNil(t, project.Description)
	assert.Equal(t, "Template description", *project.Description)

	// Verify creator added as owner
	members, _ := repo.ListMembers(context.Background(), project.ID)
	require.Len(t, members, 1)
	assert.Equal(t, creatorID, members[0].UserID)
	assert.Equal(t, models.ProjectRoleOwner, members[0].Role)
}

func TestService_CreateFromTemplate_NotATemplate(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	regularID := uuid.New()
	repo.addProject(&models.Project{
		ID:         regularID,
		Name:       "Regular",
		ProjectKey: "RG",
		IsTemplate: false,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.CreateFromTemplate(context.Background(), regularID, "New", "NW", uuid.New())

	assert.ErrorIs(t, err, ErrTemplateNotFound)
}

func TestService_CreateFromTemplate_DuplicateKey(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	templateID := uuid.New()
	repo.addProject(&models.Project{
		ID:         templateID,
		Name:       "Template",
		ProjectKey: "TM",
		IsTemplate: true,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	_, err := svc.CreateFromTemplate(context.Background(), templateID, "New", "TM", uuid.New())

	assert.ErrorIs(t, err, ErrKeyAlreadyExists)
}

// ============================================================================
// UserPreference Tests
// ============================================================================

func TestService_SetUserPreference_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	pref := &models.UserProjectPreference{
		UserID:    uuid.New(),
		ProjectID: uuid.New(),
		ViewType:  "kanban",
	}

	err := svc.SetUserPreference(context.Background(), pref)

	require.NoError(t, err)
	assert.NotZero(t, pref.UpdatedAt)
}

func TestService_SetUserPreference_InvalidViewType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	pref := &models.UserProjectPreference{
		UserID:    uuid.New(),
		ProjectID: uuid.New(),
		ViewType:  "timeline",
	}

	err := svc.SetUserPreference(context.Background(), pref)

	assert.ErrorIs(t, err, ErrInvalidViewType)
}

func TestService_GetUserPreference_Exists(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	projectID := uuid.New()

	// Set first
	err := svc.SetUserPreference(context.Background(), &models.UserProjectPreference{
		UserID:    userID,
		ProjectID: projectID,
		ViewType:  "kanban",
	})
	require.NoError(t, err)

	// Get back
	pref, err := svc.GetUserPreference(context.Background(), userID, projectID)

	require.NoError(t, err)
	require.NotNil(t, pref)
	assert.Equal(t, "kanban", pref.ViewType)
}

func TestService_GetUserPreference_NoPreference(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	pref, err := svc.GetUserPreference(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, pref)
}
