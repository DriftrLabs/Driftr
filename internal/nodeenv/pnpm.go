package nodeenv

// Config keys driftr manages on the user's global pnpm configuration.
const (
	// StoreDirKey points pnpm at driftr's shared content-addressable store.
	StoreDirKey = "store-dir"
	// GlobalVirtualStoreKey enables pnpm's global virtual store, allowing the
	// virtual store (node_modules/.pnpm) to be shared across projects. This is
	// a newer pnpm flag; older pnpm versions may not recognize it.
	GlobalVirtualStoreKey = "enable-global-virtual-store"
)

// Pnpm wraps pnpm/corepack invocations through a Runner.
type Pnpm struct {
	r Runner
}

// NewPnpm returns a Pnpm service backed by the given Runner.
func NewPnpm(r Runner) *Pnpm { return &Pnpm{r: r} }

// Installed reports whether pnpm is resolvable on PATH.
func (p *Pnpm) Installed() bool {
	_, err := p.r.LookPath("pnpm")
	return err == nil
}

// CorepackAvailable reports whether corepack is resolvable on PATH.
func (p *Pnpm) CorepackAvailable() bool {
	_, err := p.r.LookPath("corepack")
	return err == nil
}

// Version returns the installed pnpm version string.
func (p *Pnpm) Version() (string, error) {
	return p.r.Run("pnpm", "--version")
}

// CorepackEnable runs `corepack enable`.
func (p *Pnpm) CorepackEnable() error {
	_, err := p.r.Run("corepack", "enable")
	return err
}

// ConfigGet returns the global pnpm config value for key. pnpm prints
// "undefined" for unset keys; that is returned verbatim for the caller to
// interpret.
func (p *Pnpm) ConfigGet(key string) (string, error) {
	return p.r.Run("pnpm", "config", "get", key)
}

// ConfigSet writes a global pnpm config value.
func (p *Pnpm) ConfigSet(key, value string) error {
	_, err := p.r.Run("pnpm", "config", "set", key, value)
	return err
}

// StorePath returns the active pnpm store path.
func (p *Pnpm) StorePath() (string, error) {
	return p.r.Run("pnpm", "store", "path")
}

// StorePrune removes orphaned packages from the store.
func (p *Pnpm) StorePrune() (string, error) {
	return p.r.Run("pnpm", "store", "prune")
}

// Install runs `pnpm install` in the current working directory.
func (p *Pnpm) Install() (string, error) {
	return p.r.Run("pnpm", "install")
}
