# zburn v0.5.0 Manual TUI Smoke Test

Run `zburn` with no arguments to launch the TUI. Walk through each section in order.

## Store lifecycle

- [ ] Fresh start (no existing data dir): password creation prompt appears
- [ ] Enter new password + confirmation: menu appears
- [ ] Quit (`q` or Ctrl+C) and relaunch: single password prompt appears
- [ ] Enter correct password: menu loads
- [ ] Quit and relaunch, enter wrong password: error message displayed, retry prompt

## Generate identity

- [ ] Menu: select "generate identity"
- [ ] Identity view shows 10 fields with random data (name, email, phone, address, dob, password)
- [ ] Arrow keys navigate between fields
- [ ] `n` regenerates all fields with new random data
- [ ] Enter on a field copies that field's value to clipboard
- [ ] `c` copies all fields to clipboard
- [ ] `s` saves the identity to the store
- [ ] Esc returns to the main menu

## Quick email

- [ ] Menu: select "generate email"
- [ ] Flash message confirms email generated
- [ ] Returns to main menu automatically
- [ ] Clipboard contains an email matching `*@zburn.id`

## Browse identities

- [ ] Menu: select "browse identities"
- [ ] List shows the previously saved identity
- [ ] Enter on an identity opens its detail view
- [ ] Detail view shows all fields
- [ ] Esc returns to the identity list
- [ ] Esc again returns to the main menu

## Credential vault

- [ ] From identity detail view, press `v` to open credential list
- [ ] Press `a` to add a new credential
- [ ] Form shows fields: label, URL, username, password, TOTP secret, notes
- [ ] Fill in test values for all fields
- [ ] Ctrl+G generates a random password in the password field
- [ ] Save the credential (Ctrl+S or Enter on save)
- [ ] Credential appears in the list
- [ ] Enter on the credential opens it for editing
- [ ] Edit a field and save: change persists
- [ ] Add a second credential
- [ ] Delete one credential: correct one is removed, other remains

## TOTP

- [ ] Add a credential with TOTP secret `JBSWY3DPEHPK3PXP`
- [ ] TOTP field displays a rotating 6-digit code
- [ ] Code changes every 30 seconds
- [ ] Verify code against a known TOTP generator (e.g., `oathtool --totp -b JBSWY3DPEHPK3PXP`)

## Settings

- [ ] Menu: select "settings"
- [ ] Three integrations shown: Namecheap, Twilio, Gmail
- [ ] All three show "not configured" initially
- [ ] Select Namecheap: form accepts API key and username
- [ ] Enter dummy values, Ctrl+S to save: status changes to "configured"
- [ ] Select Twilio: form accepts Account SID, Auth Token, phone number
- [ ] Enter dummy values, Ctrl+S to save: status changes to "configured"
- [ ] Select Gmail: form accepts input (do NOT attempt OAuth flow)
- [ ] Quit and relaunch, enter password: settings persist with "configured" status

## Burn cascade

- [ ] Generate and save a new identity
- [ ] Open it in detail view, add 2 credentials
- [ ] Press `d` to initiate burn/delete
- [ ] Confirmation dialog shows the plan including credential count
- [ ] Press `n` to cancel: returns to detail view, nothing deleted
- [ ] Press `d` again to re-initiate burn
- [ ] Press `y` to confirm: burn executes
- [ ] Result screen shows success
- [ ] Identity is gone from the browse list
- [ ] Other previously saved identities (if any) are unaffected

## CLI (interactive, from terminal)

- [ ] `zburn identity --save` prompts for password, prints identity, saves to store
- [ ] `zburn list --json` prompts for password, outputs JSON array of saved identities
- [ ] `zburn forget <id>` prompts for password, prints deletion confirmation
- [ ] `zburn list --json` after forget: identity no longer in output
