# Development Guide

## Workflow

### Fork the main repository

1. Go to https://github.com/redhat-developer/odo
2. Click the "Fork" button (at the top right)

### Clone your fork

The commands below require that you have $GOPATH. We highly recommended you put odo code into your $GOPATH.

```sh
git clone https://github.com/$YOUR_GITHUB_USERNAME/odo.git $GOPATH/src/github.com/redhat-developer/odo
cd $GOPATH/src/github.com/redhat-developer/odo
git remote add upstream 'https://github.com/redhat-developer/odo'
```

### Create a branch and make changes

```sh
git checkout -b myfeature
# Make your code changes
```

### Keeping your development fork in sync

```sh
git fetch upstream
git rebase upstream/master
```

**Note for maintainers**: If you have write access to the main repository at github.com/redhat-developer/odo, you should modify your git configuration so that you can't accidentally push to upstream:

```sh
git remote set-url --push upstream no_push
```

### Pushing changes to your fork

```sh
git commit
git push -f origin myfeature
```

### Creating a pull request

1. Visit https://github.com/$YOUR_GITHUB_USERNAME/odo.git
2. Click the "Compare and pull request" button next to your "myfeature" branch.
3. Check out the pull request process for more details

**Pull request description:** A PR should contain an accurate description of the feature being implemented as well as a link to an active issue (if any).

### Test Driven Development

We follow Test Driven Development(TDD) workflow in our development process. You can read more about it [here](/docs/tdd-workflow.md).

### Unit tests

##### Introduction

Unit-tests for Odo functions are written using package [fake](https://godoc.org/k8s.io/client-go/kubernetes/fake). This allows us to create a fake client, and then mock the API calls defined under [OpenShift client-go](https://github.com/openshift/client-go) and [k8s client-go](https://godoc.org/k8s.io/client-go).

The tests are written in golang using the [pkg/testing](https://golang.org/pkg/testing/) package.

##### Writing unit tests

 1. Identify the APIs used by the function to be tested.

 2. Initialise the fake client along with the relevant clientsets.

 3. In the case of functions fetching or creating new objects through the APIs, add a [reactor](https://godoc.org/k8s.io/client-go/testing#Fake.AddReactor) interface returning fake objects. 

 4. Verify the objects returned

##### Initialising fake client and creating fake objects

Let us understand the initialisation of fake clients and therefore the creation of fake objects with an example.

The function `GetImageStreams` in [pkg/occlient.go](https://github.com/redhat-developer/odo/blob/master/pkg/occlient/occlient.go) fetches imagestream objects through the API:

```go
func (c *Client) GetImageStreams(namespace string) ([]imagev1.ImageStream, error) {
        imageStreamList, err := c.imageClient.ImageStreams(namespace).List(metav1.ListOptions{})
        if err != nil {
                return nil, errors.Wrap(err, "unable to list imagestreams")
        }
        return imageStreamList.Items, nil
}

```

1. For writing the tests, we start by initialising the fake client using the function `FakeNew()` which initialises the image clientset harnessed by 	`GetImageStreams` funtion:

    ```go
    client, fkclientset := FakeNew()
    ```

2. In the `GetImageStreams` funtions, the list of imagestreams is fetched through the API. While using fake client, this list can be emulated using a [`PrependReactor`](https://github.com/kubernetes/client-go/blob/master/testing/fake.go) interface:
 
   ```go
	fkclientset.ImageClientset.PrependReactor("list", "imagestreams", func(action ktesting.Action) (bool, runtime.Object, error) {
        	return true, fakeImageStreams(tt.args.name, tt.args.namespace), nil
        })
   ```

   The `PrependReactor` expects `resource` and `verb` to be passed in as arguments. We can get this information by looking at the [`List` function for fake imagestream](https://github.com/openshift/client-go/blob/master/image/clientset/versioned/typed/image/v1/fake/fake_imagestream.go):


   	```go
    func (c *FakeImageStreams) List(opts v1.ListOptions) (result *image_v1.ImageStreamList, err error) {
        	obj, err := c.Fake.Invokes(testing.NewListAction(imagestreamsResource, imagestreamsKind, c.ns, opts), &image_v1.ImageStreamList{})
		...
    }
        
    func NewListAction(resource schema.GroupVersionResource, kind schema.GroupVersionKind, namespace string, opts interface{}) ListActionImpl {
        	action := ListActionImpl{}
        	action.Verb = "list"
        	action.Resource = resource
        	action.Kind = kind
        	action.Namespace = namespace
        	labelSelector, fieldSelector, _ := ExtractFromListOptions(opts)
        	action.ListRestrictions = ListRestrictions{labelSelector, fieldSelector}

        	return action
    }
    ```


  The `List` function internally calls `NewListAction` defined in [k8s.io/client-go/testing/actions.go](https://github.com/kubernetes/client-go/blob/master/testing/actions.go).  From these functions, we see that the `resource` and `verb`to be passed into the `PrependReactor` interface are `imagestreams` and `list` respectively. 


  You can see the entire test function `TestGetImageStream` in [pkg/occlient/occlient_test.go](https://github.com/redhat-developer/odo/blob/master/pkg/occlient/occlient_test.go)

## Dependency Management

odo uses `glide` to manage dependencies.

They are not strictly required for building odo but they are required when managing dependencies under the `vendor/` directory.

If you want to make changes to dependencies please make sure that `glide` is installed and are in your `$PATH`.

### Installing glide

Get `glide`:

```sh
go get -u github.com/Masterminds/glide
```

Check that `glide` is working

```sh
glide --version
```

### Using glide to add a new dependency

#### Adding new dependency

1. Update `glide.yaml` file. Add new packages or subpackages to `glide.yaml` depending if you added whole new package as dependency or just new subpackage.

2. Run `glide update --strip-vendor` to get new dependencies

3. Commit updated `glide.yaml`, `glide.lock` and `vendor` to git.


#### Updating dependencies

1. Set new package version in  `glide.yaml` file.

2. Run `glide update --strip-vendor` to update dependencies

# Release guide

## Making a release

Making artifacts for new release is automated. 

When new git tag is created, Travis-ci deploy job automatically builds binaries and uploads it to GitHub release page.

1. Create PR with updated version in following files:

    - [cmd/version.go](/cmd/version.go)
    - [scripts/install.sh](/scripts/install.sh)
    - [README.md](/README.md)

    There is a helper script [scripts/bump-version.sh](/scripts/bump-version.sh) that should change version number in all files listed above (expect odo.rb).

    To update the CLI Structure in README.md, run `make generate-cli-structure` and update the section in [README.md](/README.md#cli-structure)

    To update the CLI reference documentation in docs/cli-reference.md, run `make generate-cli-structure > docs/cli-reference.md`.

2. Merge the above PR

3. Once the PR is merged create and push new git tag for version.
    ```
    git tag v0.0.1
    git push upstream v0.0.1
    ```
    **Or** create the new release using GitHub site (this has to be a proper release, not just draft). 

    Do not upload any binaries for release

    When new tag is created Travis-CI starts a special deploy job.

    This job builds binaries automatically (via `make prepare-release`) and then uploads it to GitHub release page (done using odo-bot user).

4. When a job finishes you should see binaries on the GitHub release page. Release is now marked as a draft. Update descriptions and publish release.

5. Verify that packages have been uploaded to rpm and deb repositories.

6. We must now update the Homebrew package. Download the current release `.tar.gz` file and retrieve the sha256 value.

    ```sh
    RELEASE=X.X.X
    wget https://github.com/redhat-developer/odo/archive/v$RELEASE.tar.gz
    sha256 v$RELEASE.tar.gz
    ```

    Then open a PR to update: [odo.rb](https://github.com/kadel/homebrew-odo/blob/master/Formula/odo.rb) in [kadel/homebrew-odo](https://github.com/kadel/homebrew-odo)

7. Confirm the binaries are available in GitHub release page.

8. Create a PR and update the file `build/VERSION` with latest version number.

## odo-bot
This is GitHub user that does all the automation.

### Scripts using odo-bot

| Script      | What it is doing                          | Access via                                    |
|-------------|-------------------------------------------|-----------------------------------------------|
| .travis.yml | Uploading binaries to GitHub release page | Personal access token `deploy-github-release` |
