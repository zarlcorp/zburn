// Package tui implements the root Bubble Tea model for zburn.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zfilesystem"
	"github.com/zarlcorp/core/pkg/zstore"
	"github.com/zarlcorp/core/pkg/zstyle"
	"github.com/zarlcorp/zburn/internal/burn"
	"github.com/zarlcorp/zburn/internal/credential"
	"github.com/zarlcorp/zburn/internal/identity"
)

type viewID int

const (
	viewPassword viewID = iota
	viewMenu
	viewGenerate
	viewList
	viewDetail
	viewCredentialList
	viewCredentialDetail
	viewCredentialForm
	viewSettings
	viewSettingsNamecheap
	viewSettingsGmail
	viewSettingsTwilio
	viewBurn
	viewForwarding
)

// ExternalServices holds optional integrations for burn cascade.
type ExternalServices struct {
	Releaser burn.PhoneReleaser
	// PhoneForIdentity returns provisioned phone config for an identity, or nil.
	PhoneForIdentity func(identityID string) *burn.PhoneConfig
}

// Model is the root TUI model.
type Model struct {
	version     string
	dataDir     string
	gen         *identity.Generator
	store       *zstore.Store
	identities  *zstore.Collection[identity.Identity]
	credentials *zstore.Collection[credential.Credential]
	configs     *zstore.Collection[configEnvelope]
	firstRun    bool
	external    ExternalServices

	active           viewID
	password         passwordModel
	menu             menuModel
	generate         generateModel
	list             listModel
	detail           detailModel
	credentialList   credentialListModel
	credentialDetail credentialDetailModel
	credentialForm   credentialFormModel
	burn             burnModel

	// settings views
	settings          settingsModel
	settingsNamecheap namecheapModel
	settingsGmail     gmailModel
	settingsTwilio    twilioModel
	forwarding        forwardingModel

	// cached config state
	ncConfig NamecheapSettings
	gmConfig GmailSettings
	twConfig TwilioSettings

	// domain rotation
	domains   []string
	domainIdx int

	// terminal dimensions
	width  int
	height int
}

// New creates the root TUI model.
func New(version, dataDir string, gen *identity.Generator, firstRun bool) Model {
	return Model{
		version:  version,
		dataDir:  dataDir,
		gen:      gen,
		firstRun: firstRun,
		active:   viewPassword,
		password: newPasswordModel(firstRun),
		menu:     newMenuModel(version),
	}
}

// SetExternalServices configures optional integrations for burn cascade.
func (m *Model) SetExternalServices(ext ExternalServices) {
	m.external = ext
}

func (m Model) Init() tea.Cmd {
	return m.password.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case passwordSubmitMsg:
		return m.openStore(msg.password)

	case navigateMsg:
		return m.navigate(msg.view)

	case saveIdentityMsg:
		return m.handleSave(msg.identity)

	case deleteIdentityMsg:
		return m.handleDelete(msg.id)

	case viewIdentityMsg:
		return m.handleViewIdentity(msg.identity)

	case viewCredentialsMsg:
		return m.loadCredentialList(msg.identity)

	case viewCredentialMsg:
		m.credentialDetail = newCredentialDetailModel(msg.credential)
		m.active = viewCredentialDetail
		return m, m.credentialDetail.Init()

	case addCredentialMsg:
		m.credentialForm = newCredentialFormModel(msg.identity, nil)
		m.active = viewCredentialForm
		return m, m.credentialForm.Init()

	case editCredentialMsg:
		c := msg.credential
		m.credentialForm = newCredentialFormModel(m.detail.identity, &c)
		m.active = viewCredentialForm
		return m, m.credentialForm.Init()

	case saveCredentialMsg:
		return m.handleSaveCredential(msg.credential)

	case deleteCredentialMsg:
		return m.handleDeleteCredential(msg.id)

	case saveNamecheapMsg:
		return m.handleSaveNamecheap(msg.settings)

	case saveGmailMsg:
		return m.handleSaveGmail(msg.settings)

	case saveTwilioMsg:
		return m.handleSaveTwilio(msg.settings)

	case disconnectGmailMsg:
		return m.handleDisconnectGmail()

	case burnStartMsg:
		return m.startBurn(msg.identity)

	case burnIdentityMsg:
		return m.executeBurn(msg.identity)

	case cycleDomainMsg:
		return m.handleCycleDomain()

	case forwardingStatusMsg:
		m.forwarding, _ = m.forwarding.Update(msg)
		return m, nil

	case forwardingResultMsg:
		return m.handleForwardingResult(msg)

	case burnResultMsg:
		m.burn, _ = m.burn.Update(msg)
		return m, clearFlashAfter3s()
	}

	return m.updateActive(msg)
}

func (m Model) View() string {
	// password and menu include the logo — render directly
	switch m.active {
	case viewPassword:
		return m.password.View()
	case viewMenu:
		return m.menu.View()
	}

	// all other views: header + separator + content + footer
	var content string
	switch m.active {
	case viewGenerate:
		content = m.generate.View()
	case viewList:
		content = m.list.View()
	case viewDetail:
		content = m.detail.View()
	case viewCredentialList:
		content = m.credentialList.View()
	case viewCredentialDetail:
		content = m.credentialDetail.View()
	case viewCredentialForm:
		content = m.credentialForm.View()
	case viewSettings:
		content = m.settings.View()
	case viewSettingsNamecheap:
		content = m.settingsNamecheap.View()
	case viewSettingsGmail:
		content = m.settingsGmail.View()
	case viewSettingsTwilio:
		content = m.settingsTwilio.View()
	case viewBurn:
		content = m.burn.View()
	case viewForwarding:
		content = m.forwarding.View()
	}

	header := zstyle.RenderHeader("zburn", viewTitle(m.active), zstyle.ZburnAccent)
	sep := zstyle.RenderSeparator(m.width)
	footer := zstyle.RenderFooter(helpFor(m.active))

	return "\n" + header + "\n" + sep + "\n" + content + "\n" + footer + "\n"
}

// viewTitle returns the display title for each view.
func viewTitle(id viewID) string {
	switch id {
	case viewGenerate:
		return "Generate Identity"
	case viewList:
		return "Saved Identities"
	case viewDetail:
		return "Identity Details"
	case viewCredentialList:
		return "Credentials"
	case viewCredentialDetail:
		return "Credential"
	case viewCredentialForm:
		return "Credential Form"
	case viewSettings:
		return "Settings"
	case viewSettingsNamecheap:
		return "Namecheap"
	case viewSettingsGmail:
		return "Gmail"
	case viewSettingsTwilio:
		return "Twilio"
	case viewBurn:
		return "Burn"
	case viewForwarding:
		return "Forwarding"
	}
	return ""
}

// helpFor returns keybinding pairs for each view's footer.
func helpFor(id viewID) []zstyle.HelpPair {
	switch id {
	case viewGenerate:
		return []zstyle.HelpPair{
			{Key: "s", Desc: "save"},
			{Key: "c", Desc: "copy all"},
			{Key: "enter", Desc: "copy field"},
			{Key: "n", Desc: "new"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewList:
		return []zstyle.HelpPair{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter", Desc: "view"},
			{Key: "d", Desc: "burn"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewDetail:
		return []zstyle.HelpPair{
			{Key: "enter", Desc: "copy field"},
			{Key: "c", Desc: "copy all"},
			{Key: "w", Desc: "credentials"},
			{Key: "d", Desc: "burn"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewCredentialList:
		return []zstyle.HelpPair{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter", Desc: "view"},
			{Key: "a", Desc: "add"},
			{Key: "d", Desc: "delete"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewCredentialDetail:
		return []zstyle.HelpPair{
			{Key: "r", Desc: "reveal"},
			{Key: "c", Desc: "copy pw"},
			{Key: "e", Desc: "edit"},
			{Key: "d", Desc: "delete"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewCredentialForm:
		return []zstyle.HelpPair{
			{Key: "tab", Desc: "next"},
			{Key: "shift+tab", Desc: "prev"},
			{Key: "space", Desc: "cycle"},
			{Key: "enter", Desc: "save"},
			{Key: "esc", Desc: "cancel"},
		}
	case viewSettings:
		return []zstyle.HelpPair{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewSettingsNamecheap:
		return []zstyle.HelpPair{
			{Key: "tab", Desc: "next"},
			{Key: "ctrl+s", Desc: "save"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewSettingsGmail:
		return []zstyle.HelpPair{
			{Key: "tab", Desc: "next"},
			{Key: "enter", Desc: "connect"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewSettingsTwilio:
		return []zstyle.HelpPair{
			{Key: "tab", Desc: "next"},
			{Key: "enter", Desc: "toggle"},
			{Key: "ctrl+s", Desc: "save"},
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	case viewBurn:
		return []zstyle.HelpPair{
			{Key: "y", Desc: "confirm"},
			{Key: "n", Desc: "cancel"},
			{Key: "q", Desc: "quit"},
		}
	case viewForwarding:
		return []zstyle.HelpPair{
			{Key: "esc", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
	}
	return nil
}

func (m Model) updateActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.active {
	case viewPassword:
		m.password, cmd = m.password.Update(msg)
	case viewMenu:
		m.menu, cmd = m.menu.Update(msg)
	case viewGenerate:
		m.generate, cmd = m.generate.Update(msg)
	case viewList:
		m.list, cmd = m.list.Update(msg)
	case viewDetail:
		m.detail, cmd = m.detail.Update(msg)
	case viewCredentialList:
		m.credentialList, cmd = m.credentialList.Update(msg)
	case viewCredentialDetail:
		m.credentialDetail, cmd = m.credentialDetail.Update(msg)
	case viewCredentialForm:
		m.credentialForm, cmd = m.credentialForm.Update(msg)
	case viewSettings:
		m.settings, cmd = m.settings.Update(msg)
	case viewSettingsNamecheap:
		m.settingsNamecheap, cmd = m.settingsNamecheap.Update(msg)
	case viewSettingsGmail:
		m.settingsGmail, cmd = m.settingsGmail.Update(msg)
	case viewSettingsTwilio:
		m.settingsTwilio, cmd = m.settingsTwilio.Update(msg)
	case viewBurn:
		m.burn, cmd = m.burn.Update(msg)
	case viewForwarding:
		m.forwarding, cmd = m.forwarding.Update(msg)
	}

	return m, cmd
}

func (m Model) openStore(password string) (tea.Model, tea.Cmd) {
	if err := os.MkdirAll(m.dataDir, 0o700); err != nil {
		m.password, _ = m.password.Update(passwordErrMsg{
			err: fmt.Errorf("create data dir: %w", err),
		})
		return m, nil
	}

	fsys := zfilesystem.NewOSFileSystem(m.dataDir)
	s, err := zstore.Open(fsys, []byte(password))
	if err != nil {
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	idCol, err := zstore.NewCollection[identity.Identity](s, "identities")
	if err != nil {
		s.Close()
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	credCol, err := zstore.NewCollection[credential.Credential](s, "credentials")
	if err != nil {
		s.Close()
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	cfgCol, err := zstore.NewCollection[configEnvelope](s, "config")
	if err != nil {
		s.Close()
		m.password, _ = m.password.Update(passwordErrMsg{err: err})
		return m, nil
	}

	m.store = s
	m.identities = idCol
	m.credentials = credCol
	m.configs = cfgCol
	m.loadConfigs()
	m.active = viewMenu
	return m, nil
}

func (m Model) navigate(view viewID) (tea.Model, tea.Cmd) {
	switch view {
	case viewMenu:
		mm := newMenuModel(m.version)
		if m.identities != nil {
			if ids, err := m.identities.List(); err == nil {
				mm.identityCount = len(ids)
			}
		}
		m.menu = mm
		m.active = viewMenu
		return m, tea.ClearScreen

	case viewGenerate:
		domain := m.currentDomain()
		id := m.gen.Generate(domain)
		m.generate = newGenerateModel(id, domain)
		m.active = viewGenerate
		return m, tea.ClearScreen

	case viewList:
		m, cmd := m.loadList()
		return m, tea.Batch(cmd, tea.ClearScreen)

	case viewDetail:
		if m.credentials != nil {
			count, err := m.countCredentials(m.detail.identity.ID)
			if err == nil {
				m.detail.credentialCount = count
			}
		}
		m.active = viewDetail
		return m, tea.ClearScreen

	case viewCredentialList:
		m, cmd := m.loadCredentialList(m.credentialList.identity)
		return m, tea.Batch(cmd, tea.ClearScreen)

	case viewSettings:
		m.settings = newSettingsModel(m.ncConfig, m.gmConfig, m.twConfig)
		m.active = viewSettings
		return m, tea.ClearScreen

	case viewSettingsNamecheap:
		m.settingsNamecheap = newNamecheapModel(m.ncConfig)
		m.active = viewSettingsNamecheap
		return m, tea.ClearScreen

	case viewSettingsGmail:
		m.settingsGmail = newGmailModel(m.gmConfig)
		m.active = viewSettingsGmail
		return m, tea.ClearScreen

	case viewSettingsTwilio:
		m.settingsTwilio = newTwilioModel(m.twConfig)
		m.active = viewSettingsTwilio
		return m, tea.ClearScreen

	case viewForwarding:
		m.forwarding = newForwardingModel(m.ncConfig, m.gmConfig)
		m.active = viewForwarding
		// only fetch if both services are configured
		if m.ncConfig.Configured() && m.gmConfig.Configured() && len(m.ncConfig.CachedDomains) > 0 {
			m.forwarding.loading = true
			return m, tea.Batch(tea.ClearScreen, fetchForwardingStatusCmd(m.ncConfig.NamecheapConfig(), m.ncConfig.CachedDomains))
		}
		return m, tea.ClearScreen

	case viewBurn:
		m.active = viewBurn
		return m, tea.ClearScreen
	}

	return m, nil
}

func (m Model) loadList() (tea.Model, tea.Cmd) {
	ids, err := m.identities.List()
	if err != nil {
		// show empty list with error flash
		m.list = newListModel(nil)
		m.list.flash = "load: " + err.Error()
		m.active = viewList
		return m, clearFlashAfter()
	}

	// sort by CreatedAt descending — zstore.List does not guarantee order
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].CreatedAt.After(ids[j].CreatedAt)
	})

	m.list = newListModel(ids)
	m.list.credCounts = m.bulkCredCounts()
	m.active = viewList
	return m, nil
}

func (m Model) bulkCredCounts() map[string]int {
	if m.credentials == nil {
		return nil
	}
	all, err := m.credentials.List()
	if err != nil {
		return nil
	}
	counts := make(map[string]int)
	for _, c := range all {
		counts[c.IdentityID]++
	}
	return counts
}

func (m Model) handleCycleDomain() (tea.Model, tea.Cmd) {
	if len(m.domains) <= 1 {
		return m, nil
	}
	m.domainIdx = (m.domainIdx + 1) % len(m.domains)
	domain := m.domains[m.domainIdx]
	id := m.generate.identity
	id.Email = m.gen.Email(id.FirstName, id.LastName, domain)
	m.generate = newGenerateModel(id, domain)
	return m, nil
}

func (m Model) handleSave(id identity.Identity) (tea.Model, tea.Cmd) {
	if err := m.identities.Put(id.ID, id); err != nil {
		m.generate.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.generate, _ = m.generate.Update(identitySavedMsg{})
	return m, clearFlashAfter()
}

func (m Model) handleDelete(id string) (tea.Model, tea.Cmd) {
	if err := m.identities.Delete(id); err != nil {
		if m.active == viewDetail {
			m.detail.flash = "delete: " + err.Error()
			return m, clearFlashAfter()
		}
		m.list.flash = "delete: " + err.Error()
		return m, clearFlashAfter()
	}

	if m.active == viewDetail {
		// go back to list after deleting from detail
		return m.loadList()
	}

	// reload list after delete
	return m.loadList()
}

func (m Model) handleViewIdentity(id identity.Identity) (tea.Model, tea.Cmd) {
	m.detail = newDetailModel(id)

	// count credentials for this identity
	if m.credentials != nil {
		count, err := m.countCredentials(id.ID)
		if err == nil {
			m.detail.credentialCount = count
		}
	}

	m.active = viewDetail
	return m, nil
}

func (m Model) countCredentials(identityID string) (int, error) {
	all, err := m.credentials.List()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range all {
		if c.IdentityID == identityID {
			count++
		}
	}
	return count, nil
}

func (m Model) loadCredentialList(id identity.Identity) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		m.credentialList = newCredentialListModel(id, nil)
		m.active = viewCredentialList
		return m, nil
	}

	all, err := m.credentials.List()
	if err != nil {
		m.credentialList = newCredentialListModel(id, nil)
		m.credentialList.flash = "load: " + err.Error()
		m.active = viewCredentialList
		return m, clearFlashAfter()
	}

	// filter by identity
	var creds []credential.Credential
	for _, c := range all {
		if c.IdentityID == id.ID {
			creds = append(creds, c)
		}
	}

	m.credentialList = newCredentialListModel(id, creds)
	m.active = viewCredentialList
	return m, nil
}

func (m Model) handleSaveCredential(c credential.Credential) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		return m, nil
	}

	if err := m.credentials.Put(c.ID, c); err != nil {
		m.credentialForm.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	// after save, go to credential detail
	m.credentialDetail = newCredentialDetailModel(c)
	m.active = viewCredentialDetail
	return m, m.credentialDetail.Init()
}

func (m Model) handleDeleteCredential(id string) (tea.Model, tea.Cmd) {
	if m.credentials == nil {
		return m, nil
	}

	if err := m.credentials.Delete(id); err != nil {
		if m.active == viewCredentialDetail {
			m.credentialDetail.flash = "delete: " + err.Error()
			return m, clearFlashAfter()
		}
		m.credentialList.flash = "delete: " + err.Error()
		return m, clearFlashAfter()
	}

	// go back to credential list; the identity is always available from
	// the credential list model since we navigate through it
	return m.loadCredentialList(m.credentialList.identity)
}

// loadConfigs reads all provider configs from the store into cached fields.
// Missing configs are silently ignored (zero value = unconfigured).
func (m *Model) loadConfigs() {
	m.ncConfig = loadConfig[NamecheapSettings](m.configs, "namecheap")
	m.gmConfig = loadConfig[GmailSettings](m.configs, "gmail")
	m.twConfig = loadConfig[TwilioSettings](m.configs, "twilio")
	m.domains = m.ncConfig.CachedDomains
	m.domainIdx = 0
}

// loadConfig reads a typed config from the envelope collection.
func loadConfig[T any](col *zstore.Collection[configEnvelope], key string) T {
	var zero T
	if col == nil {
		return zero
	}

	env, err := col.Get(key)
	if err != nil {
		return zero
	}

	var v T
	if err := json.Unmarshal(env.Data, &v); err != nil {
		return zero
	}

	return v
}

// saveConfig persists a typed config into the envelope collection.
func saveConfig[T any](col *zstore.Collection[configEnvelope], key string, v T) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return col.Put(key, configEnvelope{Data: data})
}

func (m Model) handleSaveNamecheap(s NamecheapSettings) (tea.Model, tea.Cmd) {
	if err := saveConfig(m.configs, "namecheap", s); err != nil {
		m.settingsNamecheap.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.ncConfig = s
	m.domains = s.CachedDomains
	m.domainIdx = 0

	cmds := []tea.Cmd{clearFlashAfter()}

	// trigger catch-all forwarding if Gmail is already configured
	if m.gmConfig.Configured() && len(s.CachedDomains) > 0 {
		cmds = append(cmds, forwardingCmd(s.NamecheapConfig(), s.CachedDomains, m.gmConfig.Email))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSaveGmail(s GmailSettings) (tea.Model, tea.Cmd) {
	if err := saveConfig(m.configs, "gmail", s); err != nil {
		m.settingsGmail.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.gmConfig = s
	m.settingsGmail.current = s
	m.settingsGmail.flash = "saved"

	cmds := []tea.Cmd{clearFlashAfter()}

	// trigger catch-all forwarding if Namecheap is already configured
	if m.ncConfig.Configured() && len(m.ncConfig.CachedDomains) > 0 && s.Email != "" {
		cmds = append(cmds, forwardingCmd(m.ncConfig.NamecheapConfig(), m.ncConfig.CachedDomains, s.Email))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSaveTwilio(s TwilioSettings) (tea.Model, tea.Cmd) {
	if err := saveConfig(m.configs, "twilio", s); err != nil {
		m.settingsTwilio.flash = "save: " + err.Error()
		return m, clearFlashAfter()
	}

	m.twConfig = s
	m.settingsTwilio.flash = "saved"
	return m, clearFlashAfter()
}

func (m Model) handleForwardingResult(msg forwardingResultMsg) (tea.Model, tea.Cmd) {
	flash := forwardingFlash(msg)

	switch m.active {
	case viewSettingsNamecheap:
		m.settingsNamecheap.flash = flash
	case viewSettingsGmail:
		m.settingsGmail.flash = flash
	case viewSettings:
		// no flash field on settings menu; ignore
	}

	return m, clearFlashAfter()
}

func (m Model) handleDisconnectGmail() (tea.Model, tea.Cmd) {
	m.gmConfig.Token = nil
	m.gmConfig.Email = ""
	if err := saveConfig(m.configs, "gmail", m.gmConfig); err != nil {
		m.settingsGmail.flash = "disconnect: " + err.Error()
		return m, clearFlashAfter()
	}

	m.settingsGmail.current = m.gmConfig
	m.settingsGmail.flash = "disconnected"
	return m, clearFlashAfter()
}

// currentDomain returns the currently selected domain, or "" if none configured.
func (m Model) currentDomain() string {
	if len(m.domains) == 0 {
		return ""
	}
	return m.domains[m.domainIdx]
}

// NamecheapConfigured reports whether Namecheap is configured.
func (m Model) NamecheapConfigured() bool { return m.ncConfig.Configured() }

// GmailConfigured reports whether Gmail is configured.
func (m Model) GmailConfigured() bool { return m.gmConfig.Configured() }

// TwilioConfigured reports whether Twilio is configured.
func (m Model) TwilioConfigured() bool { return m.twConfig.Configured() }

func (m Model) startBurn(id identity.Identity) (tea.Model, tea.Cmd) {
	req := m.buildBurnRequest(id)
	plan := burn.Plan(req)
	m.burn = newBurnModel(id, plan)
	m.active = viewBurn
	return m, nil
}

func (m Model) executeBurn(id identity.Identity) (tea.Model, tea.Cmd) {
	req := m.buildBurnRequest(id)
	return m, func() tea.Msg {
		result := burn.Execute(context.Background(), req)
		return burnResultMsg{result: result}
	}
}

func (m Model) buildBurnRequest(id identity.Identity) burn.Request {
	req := burn.Request{
		Identity:    id,
		Credentials: credentialStoreOrEmpty(m.credentials),
		Identities:  identityStoreOrEmpty(m.identities),
	}

	// phone release — configured when we have a releaser and a lookup func
	if m.external.Releaser != nil && m.external.PhoneForIdentity != nil {
		if phone := m.external.PhoneForIdentity(id.ID); phone != nil {
			req.Phone = phone
			req.Releaser = m.external.Releaser
		}
	}

	return req
}

// credentialStoreOrEmpty returns the collection as a burn.CredentialStore,
// or a no-op store if the collection is nil (store not yet opened).
func credentialStoreOrEmpty(col *zstore.Collection[credential.Credential]) burn.CredentialStore {
	if col == nil {
		return emptyCredentialStore{}
	}
	return col
}

// identityStoreOrEmpty returns the collection as a burn.IdentityStore,
// or a no-op store if the collection is nil (store not yet opened).
func identityStoreOrEmpty(col *zstore.Collection[identity.Identity]) burn.IdentityStore {
	if col == nil {
		return emptyIdentityStore{}
	}
	return col
}

type emptyCredentialStore struct{}

func (emptyCredentialStore) List() ([]credential.Credential, error) { return nil, nil }
func (emptyCredentialStore) Delete(string) error                    { return nil }

type emptyIdentityStore struct{}

func (emptyIdentityStore) Delete(string) error { return nil }

// Close cleans up resources. Call after the program exits.
func (m Model) Close() {
	if m.store != nil {
		m.store.Close()
	}
}
