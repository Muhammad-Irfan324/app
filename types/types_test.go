package types

import (
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/docker/app/internal"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

const (
	validMeta = `name: test-app
version: 0.1.0`
	validCompose = `version: "3.0"
services:
  web:
    image: nginx`
	validParameters = `foo: bar`
)

func TestNewApp(t *testing.T) {
	app, err := NewApp("any-app")
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(app.Path, "any-app"))
}

func TestNewAppFromDefaultFiles(t *testing.T) {
	dir := fs.NewDir(t, "my-app",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.ParametersFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ParametersRaw(), 1))
	assertContentIs(t, app.ParametersRaw()[0], `foo: bar`)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
	assertContentIs(t, app.MetadataRaw(), validMeta)
}

func TestNewAppWithOpError(t *testing.T) {
	_, err := NewApp("any-app", func(_ *App) error { return errors.New("error creating") })
	assert.ErrorContains(t, err, "error creating")
}

func TestWithPath(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithPath("any-path")(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(app.Path, "any-path"))
}

func TestWithCleanup(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithCleanup(func() {})(app)
	assert.NilError(t, err)
	assert.Assert(t, app.Cleanup != nil)
}

func TestWithParametersFilesError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithParametersFiles("any-parameters-file")(app)
	assert.ErrorContains(t, err, "open any-parameters-file")
}

func TestWithParametersFiles(t *testing.T) {
	dir := fs.NewDir(t, "parameters",
		fs.WithFile("my-parameters-file", validParameters),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithParametersFiles(dir.Join("my-parameters-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ParametersRaw(), 1))
	assertContentIs(t, app.ParametersRaw()[0], validParameters)
}

func TestWithParameters(t *testing.T) {
	r := strings.NewReader(validParameters)
	app := &App{Path: "my-app"}
	err := WithParameters(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ParametersRaw(), 1))
	assertContentIs(t, app.ParametersRaw()[0], validParameters)
}

func TestWithComposeFilesError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithComposeFiles("any-compose-file")(app)
	assert.ErrorContains(t, err, "open any-compose-file")
}

func TestWithComposeFiles(t *testing.T) {
	dir := fs.NewDir(t, "composes",
		fs.WithFile("my-compose-file", validCompose),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithComposeFiles(dir.Join("my-compose-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
}

func TestWithComposes(t *testing.T) {
	r := strings.NewReader(validCompose)
	app := &App{Path: "my-app"}
	err := WithComposes(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
}

func TestMetadataFileError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := MetadataFile("any-metadata-file")(app)
	assert.ErrorContains(t, err, "open any-metadata-file")
}

func TestMetadataFile(t *testing.T) {
	dir := fs.NewDir(t, "metadata",
		fs.WithFile("my-metadata-file", validMeta),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := MetadataFile(dir.Join("my-metadata-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, app.MetadataRaw() != nil)
	assertContentIs(t, app.MetadataRaw(), validMeta)
}

func TestMetadata(t *testing.T) {
	r := strings.NewReader(validMeta)
	app := &App{Path: "my-app"}
	err := Metadata(r)(app)
	assert.NilError(t, err)
	assertContentIs(t, app.MetadataRaw(), validMeta)
}

func assertContentIs(t *testing.T, data []byte, expected string) {
	t.Helper()
	assert.Assert(t, is.Equal(string(data), expected))
}

func TestWithAttachmentsAndNestedDirectories(t *testing.T) {
	dir := fs.NewDir(t, "externalfile",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.ParametersFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithFile("config.cfg", "something"),
		fs.WithDir("nesteddirectory",
			fs.WithFile("nestedconfig.cfg", "something"),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Attachments(), 2))
	assert.Equal(t, app.Attachments()[0].Path(), "config.cfg")
	assert.Equal(t, app.Attachments()[0].Size(), int64(9))
	assert.Equal(t, app.Attachments()[1].Path(), "nesteddirectory/nestedconfig.cfg")
}

func TestAttachmentsAreSorted(t *testing.T) {
	dir := fs.NewDir(t, "externalfile",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.ParametersFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithFile("c.cfg", "something"),
		fs.WithFile("a.cfg", "something"),
		fs.WithFile("b.cfg", "something"),
		fs.WithDir("nesteddirectory",
			fs.WithFile("a.cfg", "something"),
			fs.WithFile("c.cfg", "something"),
			fs.WithFile("b.cfg", "something"),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Attachments(), 6))
	assert.Equal(t, app.Attachments()[0].Path(), "a.cfg")
	assert.Equal(t, app.Attachments()[1].Path(), "b.cfg")
	assert.Equal(t, app.Attachments()[2].Path(), "c.cfg")
	assert.Equal(t, app.Attachments()[3].Path(), "nesteddirectory/a.cfg")
	assert.Equal(t, app.Attachments()[4].Path(), "nesteddirectory/b.cfg")
	assert.Equal(t, app.Attachments()[5].Path(), "nesteddirectory/c.cfg")
}

func TestWithAttachmentsIncludingNestedCoreFiles(t *testing.T) {
	dir := fs.NewDir(t, "attachments",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.ParametersFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithDir("nesteddirectory",
			fs.WithFile(internal.MetadataFileName, validMeta),
			fs.WithFile(internal.ParametersFileName, `foo: bar`),
			fs.WithFile(internal.ComposeFileName, validCompose),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Attachments(), 3))
	assert.Equal(t, app.Attachments()[0].Path(), path.Join("nesteddirectory", internal.ComposeFileName))
	assert.Equal(t, app.Attachments()[1].Path(), path.Join("nesteddirectory", internal.MetadataFileName))
	assert.Equal(t, app.Attachments()[2].Path(), path.Join("nesteddirectory", internal.ParametersFileName))
}

func TestValidateBrokenMetadata(t *testing.T) {
	r := strings.NewReader(`#version: 0.1.0-missing
name: MustBeAValidUntaggedRegistryReferenceButNotEvaluatedByTheSchema
maintainers:
    - name: user
      email: user@email.com
    - name: user2
    - name: bad-user
      email: bad-email
unknown: property`)
	app := &App{Path: "my-app"}
	err := Metadata(r)(app)
	assert.Error(t, err, `failed to validate metadata:
- (root): version is required
- maintainers.2.email: Does not match format 'email'`)
}

func TestValidateBrokenParameters(t *testing.T) {
	metadata := strings.NewReader(`version: "0.1"
name: myname`)
	composeFile := strings.NewReader(`version: "3.6"`)
	brokenParameters := strings.NewReader(`my-parameters:
    1: toto`)
	app := &App{Path: "my-app"}
	err := Metadata(metadata)(app)
	assert.NilError(t, err)
	err = WithComposes(composeFile)(app)
	assert.NilError(t, err)
	err = WithParameters(brokenParameters)(app)
	assert.ErrorContains(t, err, `Non-string key in my-parameters: 1`)
}
