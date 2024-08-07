package repo

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-base32"
	"github.com/multiformats/go-multiaddr"

	"github.com/BurntSushi/toml"

	fslock "github.com/ipfs/go-fs-lock"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/Filecoin-Titan/titan-container/node/config"

	"github.com/Filecoin-Titan/titan-container/node/fsutil"
	"github.com/Filecoin-Titan/titan-container/node/types"
)

const (
	fsAPI        = "api"
	fsAPIToken   = "token"
	fsConfig     = "config.toml"
	fsDatastore  = "datastore"
	fsLock       = "repo.lock"
	fsKeystore   = "keystore"
	fsPrivateKey = "private.key"
	fsUUID       = "uuid"
	fsCert       = "cert"
	fsCAKey      = "titannet.io.key"
	fsCACrt      = "titannet.io.crt"
	fsDeployID   = "deployment.id"
)

func NewRepoTypeFromString(t string) RepoType {
	switch t {
	case "Manager":
		return Manager
	case "Provider":
		return Provider
	default:
		panic("unknown RepoType")
	}
}

type RepoType interface {
	Type() string
	Config() interface{}

	// APIFlags returns flags passed on the command line with the listen address
	// of the API server (only used by the tests), in the order of precedence they
	// should be applied for the requested kind of node.
	APIFlags() []string

	RepoFlags() []string

	// APIInfoEnvVars returns the environment variables to use in order of precedence
	// to determine the API endpoint of the specified node type.
	//
	// It returns the current variables and deprecated ones separately, so that
	// the user can log a warning when deprecated ones are found to be in use.
	APIInfoEnvVars() (string, []string, []string)
}

type manager struct{}

var Manager manager

func (manager) Type() string {
	return "Manager"
}

func (manager) Config() interface{} {
	return config.DefaultManagerCfg()
}

func (manager) APIFlags() []string {
	return []string{"manager-api-url"}
}

func (manager) RepoFlags() []string {
	return []string{"manager-repo"}
}

func (manager) APIInfoEnvVars() (primary string, fallbacks []string, deprecated []string) {
	return "MANAGER_API_INFO", nil, nil
}

type provider struct{}

var Provider provider

func (provider) Type() string {
	return "Edge"
}

func (provider) Config() interface{} {
	return config.DefaultProviderCfg()
}

func (provider) APIFlags() []string {
	return []string{"provider-api-url"}
}

func (provider) RepoFlags() []string {
	return []string{"provider-repo"}
}

func (provider) APIInfoEnvVars() (primary string, fallbacks []string, deprecated []string) {
	return "PROVIDER_API_INFO", nil, nil
}

var log = logging.Logger("repo")

var ErrRepoExists = xerrors.New("repo exists")

// FsRepo is struct for repo, use NewFS to create
type FsRepo struct {
	path       string
	configPath string
}

var _ Repo = &FsRepo{}

// NewFS creates a repo instance based on a path on file system
func NewFS(path string) (*FsRepo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path:       path,
		configPath: filepath.Join(path, fsConfig),
	}, nil
}

func (fsr *FsRepo) SetConfigPath(cfgPath string) {
	fsr.configPath = cfgPath
}

func (fsr *FsRepo) Exists() (bool, error) {
	_, err := os.Stat(filepath.Join(fsr.path, fsDatastore))
	notexist := os.IsNotExist(err)
	if notexist {
		_, err = os.Stat(filepath.Join(fsr.path, fsKeystore))
		notexist = os.IsNotExist(err)
		if notexist {
			err = nil
		}
	}
	return !notexist, err
}

func (fsr *FsRepo) Init(t RepoType) error {
	exist, err := fsr.Exists()
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	log.Infof("Initializing repo at '%s'", fsr.path)
	err = os.MkdirAll(fsr.path, 0o755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := fsr.initConfig(t); err != nil {
		return xerrors.Errorf("init config: %w", err)
	}

	return fsr.initKeystore()
}

func (fsr *FsRepo) initConfig(t RepoType) error {
	_, err := os.Stat(fsr.configPath)
	if err == nil {
		// exists
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	c, err := os.Create(fsr.configPath)
	if err != nil {
		return err
	}

	comm, err := config.GenerateConfigComment(t.Config())
	if err != nil {
		return xerrors.Errorf("comment: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		return xerrors.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return xerrors.Errorf("close config: %w", err)
	}
	return nil
}

func (fsr *FsRepo) initKeystore() error {
	kstorePath := filepath.Join(fsr.path, fsKeystore)
	if _, err := os.Stat(kstorePath); err == nil {
		return ErrRepoExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(kstorePath, 0o700)
}

// APIEndpoint returns endpoint of API in this repo
func (fsr *FsRepo) APIEndpoint() (multiaddr.Multiaddr, error) {
	p := filepath.Join(fsr.path, fsAPI)

	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, xerrors.Errorf("failed to read %q: %w", p, err)
	}
	strma := string(data)
	strma = strings.TrimSpace(strma)

	apima, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, err
	}
	return apima, nil
}

func (fsr *FsRepo) APIToken() ([]byte, error) {
	p := filepath.Join(fsr.path, fsAPIToken)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

func (fsr *FsRepo) PrivateKey() ([]byte, error) {
	p := filepath.Join(fsr.path, fsPrivateKey)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		return nil, ErrNoAPIEndpoint
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

func (fsr *FsRepo) UUID() ([]byte, error) {
	p := filepath.Join(fsr.path, fsUUID)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		return nil, ErrNoUUID
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

// Lock acquires exclusive lock on this repo
func (fsr *FsRepo) Lock(repoType RepoType) (LockedRepo, error) {
	locked, err := fslock.Locked(fsr.path, fsLock)
	if err != nil {
		return nil, xerrors.Errorf("could not check lock status: %w", err)
	}
	if locked {
		return nil, ErrRepoAlreadyLocked
	}

	closer, err := fslock.Lock(fsr.path, fsLock)
	if err != nil {
		return nil, xerrors.Errorf("could not lock the repo: %w", err)
	}
	return &fsLockedRepo{
		path:       fsr.path,
		configPath: fsr.configPath,
		repoType:   repoType,
		closer:     closer,
	}, nil
}

// Like Lock, except datastores will work in read-only mode
func (fsr *FsRepo) LockRO(repoType RepoType) (LockedRepo, error) {
	lr, err := fsr.Lock(repoType)
	if err != nil {
		return nil, err
	}

	lr.(*fsLockedRepo).readonly = true
	return lr, nil
}

func (fsr *FsRepo) DeploymentID() ([]byte, error) {
	p := filepath.Join(fsr.path, fsDeployID)
	f, err := os.Open(p)

	if os.IsNotExist(err) {
		return nil, ErrNoDeployment
	} else if err != nil {
		return nil, err
	}
	defer f.Close() //nolint: errcheck // Read only op

	tb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(tb), nil
}

func (fsr *FsRepo) SetDeploymentID(id []byte) error {
	return os.WriteFile(filepath.Join(fsr.path, fsDeployID), id, 0o600)
}

type fsLockedRepo struct {
	path       string
	configPath string
	repoType   RepoType
	closer     io.Closer
	readonly   bool

	ds     map[string]datastore.Batching
	dsErr  error
	dsOnce sync.Once

	ssPath string
	ssErr  error
	ssOnce sync.Once

	storageLk sync.Mutex
	configLk  sync.Mutex
}

func (fsr *fsLockedRepo) RepoType() RepoType {
	return fsr.repoType
}

func (fsr *fsLockedRepo) Readonly() bool {
	return fsr.readonly
}

func (fsr *fsLockedRepo) Path() string {
	return fsr.path
}

func (fsr *fsLockedRepo) Close() error {
	err := os.Remove(fsr.join(fsAPI))

	if err != nil && !os.IsNotExist(err) {
		return xerrors.Errorf("could not remove API file: %w", err)
	}
	if fsr.ds != nil {
		for _, ds := range fsr.ds {
			if err := ds.Close(); err != nil {
				return xerrors.Errorf("could not close datastore: %w", err)
			}
		}
	}

	err = fsr.closer.Close()
	fsr.closer = nil
	return err
}

func (fsr *fsLockedRepo) SplitstorePath() (string, error) {
	fsr.ssOnce.Do(func() {
		path := fsr.join(filepath.Join(fsDatastore, "splitstore"))

		if err := os.MkdirAll(path, 0o755); err != nil {
			fsr.ssErr = err
			return
		}

		fsr.ssPath = path
	})

	return fsr.ssPath, fsr.ssErr
}

// join joins path elements with fsr.path
func (fsr *fsLockedRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}

func (fsr *fsLockedRepo) stillValid() error {
	if fsr.closer == nil {
		return ErrClosedRepo
	}
	return nil
}

func (fsr *fsLockedRepo) Config() (interface{}, error) {
	fsr.configLk.Lock()
	defer fsr.configLk.Unlock()

	return fsr.loadConfigFromDisk()
}

func (fsr *fsLockedRepo) loadConfigFromDisk() (interface{}, error) {
	return config.FromFile(fsr.configPath, fsr.repoType.Config())
}

func (fsr *fsLockedRepo) SetConfig(c func(interface{})) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	fsr.configLk.Lock()
	defer fsr.configLk.Unlock()

	cfg, err := fsr.loadConfigFromDisk()
	if err != nil {
		return err
	}
	// mutate in-memory representation of config
	c(cfg)

	// buffer into which we write TOML bytes
	buf := new(bytes.Buffer)

	// encode now-mutated config as TOML and write to buffer
	err = toml.NewEncoder(buf).Encode(cfg)
	if err != nil {
		return err
	}
	// write buffer of TOML bytes to config file
	err = os.WriteFile(fsr.configPath, buf.Bytes(), 0o644)
	if err != nil {
		return err
	}

	return nil
}

func (fsr *fsLockedRepo) Stat(path string) (fsutil.FsStat, error) {
	return fsutil.Statfs(path)
}

func (fsr *fsLockedRepo) DiskUsage(path string) (int64, error) {
	si, err := fsutil.FileSize(path)
	if err != nil {
		return 0, err
	}
	return si.OnDisk, nil
}

func (fsr *fsLockedRepo) SetAPIEndpoint(ma multiaddr.Multiaddr) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsAPI), []byte(ma.String()), 0o644)
}

func (fsr *fsLockedRepo) SetAPIToken(token []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsAPIToken), token, 0o600)
}

func (fsr *fsLockedRepo) SetPrivateKey(key []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsPrivateKey), key, 0o600)
}

func (fsr *fsLockedRepo) KeyStore() (types.KeyStore, error) {
	if err := fsr.stillValid(); err != nil {
		return nil, err
	}
	return fsr, nil
}

func (fsr *fsLockedRepo) SetUUID(uuid []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}
	return os.WriteFile(fsr.join(fsUUID), uuid, 0o600)
}

var kstrPermissionMsg = "permissions of key: '%s' are too relaxed, " +
	"required: 0600, got: %#o"

// List lists all the keys stored in the KeyStore
func (fsr *fsLockedRepo) List() ([]string, error) {
	if err := fsr.stillValid(); err != nil {
		return nil, err
	}

	kstorePath := fsr.join(fsKeystore)
	dir, err := os.Open(kstorePath)
	if err != nil {
		return nil, xerrors.Errorf("opening dir to list keystore: %w", err)
	}
	defer dir.Close() //nolint:errcheck
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, xerrors.Errorf("reading keystore dir: %w", err)
	}
	keys := make([]string, 0, len(files))
	for _, f := range files {
		if f.Mode()&0o077 != 0 {
			return nil, xerrors.Errorf(kstrPermissionMsg, f.Name(), f.Mode())
		}
		name, err := base32.RawStdEncoding.DecodeString(f.Name())
		if err != nil {
			return nil, xerrors.Errorf("decoding key: '%s': %w", f.Name(), err)
		}
		keys = append(keys, string(name))
	}
	return keys, nil
}

// Get gets a key out of keystore and returns types.KeyInfo coresponding to named key
func (fsr *fsLockedRepo) Get(name string) (types.KeyInfo, error) {
	if err := fsr.stillValid(); err != nil {
		return types.KeyInfo{}, err
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return types.KeyInfo{}, xerrors.Errorf("opening key '%s': %w", name, types.ErrKeyInfoNotFound)
	} else if err != nil {
		return types.KeyInfo{}, xerrors.Errorf("opening key '%s': %w", name, err)
	}

	// if fstat.Mode()&0o077 != 0 {
	// 	return types.KeyInfo{}, xerrors.Errorf(kstrPermissionMsg, name, fstat.Mode())
	// }

	file, err := os.Open(keyPath)
	if err != nil {
		return types.KeyInfo{}, xerrors.Errorf("opening key '%s': %w", name, err)
	}
	defer file.Close() //nolint: errcheck // read only op

	data, err := io.ReadAll(file)
	if err != nil {
		return types.KeyInfo{}, xerrors.Errorf("reading key '%s': %w", name, err)
	}

	var res types.KeyInfo
	err = json.Unmarshal(data, &res)
	if err != nil {
		return types.KeyInfo{}, xerrors.Errorf("decoding key '%s': %w", name, err)
	}

	return res, nil
}

const KTrashPrefix = "trash-"

// Put saves key info under given name
func (fsr *fsLockedRepo) Put(name string, info types.KeyInfo) error {
	return fsr.put(name, info, 0)
}

func (fsr *fsLockedRepo) put(rawName string, info types.KeyInfo, retries int) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	name := rawName
	if retries > 0 {
		name = fmt.Sprintf("%s-%d", rawName, retries)
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	_, err := os.Stat(keyPath)
	if err == nil && strings.HasPrefix(name, KTrashPrefix) {
		// retry writing the trash-prefixed file with a number suffix
		return fsr.put(rawName, info, retries+1)
	} else if err == nil {
		return xerrors.Errorf("checking key before put '%s': %w", name, types.ErrKeyExists)
	} else if !os.IsNotExist(err) {
		return xerrors.Errorf("checking key before put '%s': %w", name, err)
	}

	keyData, err := json.Marshal(info)
	if err != nil {
		return xerrors.Errorf("encoding key '%s': %w", name, err)
	}

	err = os.WriteFile(keyPath, keyData, 0o600)
	if err != nil {
		return xerrors.Errorf("writing key '%s': %w", name, err)
	}
	return nil
}

func (fsr *fsLockedRepo) Delete(name string) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	encName := base32.RawStdEncoding.EncodeToString([]byte(name))
	keyPath := fsr.join(fsKeystore, encName)

	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return xerrors.Errorf("checking key before delete '%s': %w", name, types.ErrKeyInfoNotFound)
	} else if err != nil {
		return xerrors.Errorf("checking key before delete '%s': %w", name, err)
	}

	err = os.Remove(keyPath)
	if err != nil {
		return xerrors.Errorf("deleting key '%s': %w", name, err)
	}
	return nil
}

func (fsr *fsLockedRepo) Certificate() ([]byte, []byte, error) {
	pc := filepath.Join(fsr.path, filepath.Join(fsCert, fsCACrt))
	fc, err := os.Open(pc)

	if os.IsNotExist(err) {
		return nil, nil, ErrNoCertificate
	} else if err != nil {
		return nil, nil, err
	}
	defer fc.Close() //nolint: errcheck // Read only op

	cert, err := io.ReadAll(fc)
	if err != nil {
		return nil, nil, err
	}

	pk := filepath.Join(fsr.path, filepath.Join(fsCert, fsCAKey))
	fk, err := os.Open(pk)

	if os.IsNotExist(err) {
		return nil, nil, ErrNoCertificate
	} else if err != nil {
		return nil, nil, err
	}
	defer fk.Close() //nolint: errcheck // Read only op
	key, err := io.ReadAll(fk)
	if err != nil {
		return nil, nil, err
	}

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, nil, err
	}

	if len(cert) == 0 {
		return nil, nil, err
	}

	parsedCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	if parsedCert.NotAfter.Before(time.Now()) {
		return nil, nil, ErrCertificateExpired
	}

	return cert, key, nil
}

func (fsr *fsLockedRepo) SetCertificate(crt, key []byte) error {
	if err := fsr.stillValid(); err != nil {
		return err
	}

	if err := os.MkdirAll(fsr.join(fsCert), 0755); err != nil {
		return fmt.Errorf("failed to mk directory: %w", err)
	}

	if err := os.WriteFile(fsr.join(filepath.Join(fsCert, fsCACrt)), crt, 0o600); err != nil {
		return err
	}

	if err := os.WriteFile(fsr.join(filepath.Join(fsCert, fsCAKey)), key, 0o600); err != nil {
		return err
	}

	return nil
}
