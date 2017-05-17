# Relevant git config options:
```
push.gpgSign
 May be set to a boolean value, or the string if-asked. A true value causes all pushes to be GPG signed, as if --signed is passed to git-push(1). The string if-asked causes
 pushes to be signed if the server supports it, as if --signed=if-asked is passed to git push. A false value may override a value from a lower-priority config file. An explicit
 command-line flag always overrides this config option.

gpg.program
 Use this custom program instead of "gpg" found on $PATH when making or verifying a PGP signature. The program must support the same command-line interface as GPG, namely, to
 verify a detached signature, "gpg --verify $file - <$signature" is run, and the program is expected to signal a good signature by exiting with code 0, and to generate an
 ASCII-armored detached signature, the standard input of "gpg -bsau $key" is fed with the contents to be signed, and the program is expected to send the result to its standard
 output.

user.signingKey
 If git-tag(1) or git-commit(1) is not selecting the key you want it to automatically when creating a signed tag or commit, you can override the default selection with this
 variable. This option is passed unchanged to gpg's --local-user parameter, so you may specify a key using any method that gpg supports.

commit.gpgSign
 A boolean to specify whether all commits should be GPG signed. Use of this option when doing operations such as rebase can result in a large number of commits being signed. It
 may be convenient to use an agent to avoid typing your GPG passphrase several times.
```

# Integration options
- custom binary replacing `gpg` with stdio interface, just modify ~/.gitconfig
	- route verification requests to original `gpg` binary if installed?
	- need `export GPG_TTY=$(tty)` in order to read `krgpg` stdout
	- example input to `krgpg`
		args: --status-fd=2 -bsau C2E6E330

		stdin:
		tree 6b7257ab742539c9e0a1372ee4d7eb19de473200
		parent 9d66027f0cbff220fdce8c2f9ab61aad4c65d6fa
		author Kevin King <4kevinking@gmail.com> 1495001712 -0400
		committer Kevin King <4kevinking@gmail.com> 1495001712 -0400

		test

- custom agent that falls back to old gpg agent, modify ~/.gnupg/gpg-agent.conf


