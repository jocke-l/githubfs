package githubfuse

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fuseutil"

	"githubfs/internal/github"
)

func convertToFuseNode(entity github.GithubRepoEntity, inode uint64) fs.Node {
	switch entity.Type {
	case "dir":
		return GithubDir{inode: inode, entity: entity}
	case "file":
		return GithubFile{inode: inode, entity: entity}
	default:
		return nil
	}
}

type GithubFS struct {
	repo string
}

func (ghfs GithubFS) Root() (fs.Node, error) {
	return convertToFuseNode(github.GetRepoRoot(ghfs.repo), 1), nil
}

type GithubDir struct {
	inode  uint64
	entity github.GithubRepoEntity
}

func (ghdir GithubDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = ghdir.inode
	attr.Mode = os.ModeDir | 0o555

	return nil
}

func (ghdir GithubDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	entities, err := ghdir.entity.ListDir()
	if err != nil {
		return nil, fuse.ToErrno(err)
	}

	for _, entity := range entities {
		if entity.Name == name {
			return convertToFuseNode(
				entity,
				fs.GenerateDynamicInode(ghdir.inode, entity.Name),
			), nil
		}
	}

	return nil, syscall.ENOENT
}

func (ghdir GithubDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entities, err := ghdir.entity.ListDir()
	if err != nil {
		return nil, fuse.ToErrno(err)
	}

	var dirents []fuse.Dirent
	for _, entity := range entities {
		var direntType fuse.DirentType
		switch entity.Type {
		case "dir":
			direntType = fuse.DT_Dir
		case "symlink":
			direntType = fuse.DT_Link
		case "file":
			direntType = fuse.DT_File
		default:
			direntType = fuse.DT_Unknown
		}

		dirents = append(
			dirents,
			fuse.Dirent{
				Inode: fs.GenerateDynamicInode(ghdir.inode, entity.Name),
				Name:  entity.Name,
				Type:  direntType,
			},
		)
	}

	return dirents, nil
}

type GithubFile struct {
	inode  uint64
	entity github.GithubRepoEntity
}

func (ghfile GithubFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	data, err := ghfile.entity.GetFile()
	if err != nil {
		return fuse.ToErrno(err)
	}

	attr.Inode = ghfile.inode
	attr.Mode = 0o444
	attr.Size = uint64(len(data))

	return nil
}

func (ghfile GithubFile) Open(
	ctx context.Context,
	req *fuse.OpenRequest,
	resp *fuse.OpenResponse,
) (fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, syscall.EACCES
	}

	return ghfile, nil
}

func (ghfile GithubFile) Read(
	ctx context.Context,
	req *fuse.ReadRequest,
	resp *fuse.ReadResponse,
) error {
	data, err := ghfile.entity.GetFile()
	if err != nil {
		return fuse.ToErrno(err)
	}

	fuseutil.HandleRead(req, resp, data)

	return nil
}

func Serve(repo string, mountpoint string) error {
	conn, err := fuse.Mount(
		mountpoint,
		fuse.FSName("githubfs"),
		fuse.ReadOnly(),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer fuse.Unmount(mountpoint)

	signalTrap := make(chan os.Signal, 1)
	signal.Notify(signalTrap, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		signal := <-signalTrap
		fmt.Println()
		fmt.Println(signal, "recieved")
		fmt.Println("Unmounting", mountpoint)
		fuse.Unmount(mountpoint)
	}()

	return fs.Serve(conn, GithubFS{repo: repo})
}
