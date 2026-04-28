module github.com/danieljustus/OpenPass

go 1.26.2

// Deferred major updates (intentionally deferred):
// - github.com/ProtonMail/go-crypto: v1.1.6 -> v1.4.1 (major version bump, potential breaking changes in crypto APIs)
// - github.com/golang/protobuf: v1.5.4 (deprecated, migration to google.golang.org/protobuf needed)
//
// RATIONALE FOR DEFERRALS (2026-04-20 audit):
// Both deferred dependencies are transitive-only — no direct imports exist in OpenPass source code.
// OpenPass → filippo.io/age → (no ProtonMail dep) | OpenPass → go-git → ProtonMail/go-crypto
// OpenPass → go-git → groupcache → golang/protobuf (deprecated)
// Upgrade risk is contained in upstream libraries (go-git, groupcache), not OpenPass itself.
// Recommendation: Keep deferrals until upstream breaks the cycle or provides migration path.
// NOTE: google.golang.org/protobuf v1.36.7 is already present as transitive dep of golang/protobuf.
// The golang/protobuf deferral may become irrelevant as groupcache migrates off it.

require (
	filippo.io/age v1.3.1
	github.com/atotto/clipboard v0.1.4
	github.com/go-git/go-git/v5 v5.18.0
	github.com/mattn/go-tty v0.0.8
	github.com/prometheus/client_golang v1.23.2
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/zalando/go-keyring v0.2.8
	golang.org/x/crypto v0.50.0
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	golang.org/x/term v0.42.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/hpke v0.4.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/elazarl/goproxy v1.8.3 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.8.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.6.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.21 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/gomega v1.39.1 // indirect
	github.com/pjbgf/sha1cd v0.5.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/skeema/knownhosts v1.3.2 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
