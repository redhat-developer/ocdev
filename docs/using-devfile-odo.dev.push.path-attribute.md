---
# Page settings
layout: default
keywords:
comments: false

# Hero section
title: Using the odo.dev.push.path related attributes
description: Push only the specified files and folders to the component.

# Micro navigation
micro_nav: true
---
`odo` uses the `odo.dev.push.path` related attribute from the devfile’s run commands to push only the specified files and folders to the component. The format of the attribute is `"odo.dev.push.path:<local_relative_path>": "<remote_relative_path>"`. We can mention multiple such attributes in the run command’s `attributes` section.

``` yaml
commands:
  - id: dev-run
    attributes:
      "dev.odo.push.path:target/quarkus-app": "remote-target/quarkus-app"
      "dev.odo.push.path:README.txt": "docs/README.txt"
    exec:
      component: tools
      commandLine: "java -jar remote-target/quarkus-app/quarkus-run.jar -Dquarkus.http.host=0.0.0.0"
      hotReloadCapable: true
      group:
        kind: run
        isDefault: true
      workingDir: $PROJECTS_ROOT
  - id: dev-debug
    exec:
      component: tools
      commandLine: "java -Xdebug -Xrunjdwp:server=y,transport=dt_socket,address=${DEBUG_PORT},suspend=n -jar remote-target/quarkus-app/quarkus-run.jar -Dquarkus.http.host=0.0.0.0"
      hotReloadCapable: true
      group:
        kind: debug
        isDefault: true
      workingDir: $PROJECTS_ROOT
```

In the above example the contents of the `quarkus-app` folder, which is inside the `target` folder, will be pushed to the remote location of `remote-target/quarkus-app` and the file `README.txt` will be pushed to `doc/README.txt`. The local path is relative to the component’s local folder. The remote location is relative to the folder containing the component’s source code inside the container.
