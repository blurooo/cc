# Plugin

## ProtoBuf

```protobuf
syntax = "proto3";

// defined the plugin
message Plugin {
  Dependency dep = 1;
  Resource resource = 2;
  Enter enter = 3;
}

// defined dependencies for tool
message Dependency {

  // You can run this plugin command while the tool is running
  message Plugin {

    message RepoFile {
      string url = 1;
      string ref = 2;
      string path = 3;
    }

    // By default, the file name is used. For example, the file name dep/go.yaml is go.
    optional string name = 1;
    oneof type {
      string file = 2;
      RepoFile repo_file = 3;
    }

  }

  repeated Plugin plugins = 1;

}

// defined resources for tool
// will be retrieved and placed in the tool's directory before installation.
message Resource {

  message Repo {
    string url = 1;
    // default master
    optional string ref = 2;
    // default resources
    optional string path = 3;
  }

  message Mirror {
    // the priorities are as follows
    // 1. os.arch
    // 2. os
    // 3. *
    map<string, string> url = 1;
    // default resources
    optional string path = 2;
    // default true
    optional bool executable = 3;
  }

  message Archive {
    // the same as mirror.url
    map<string, string> url = 1;
    // default resources
    optional string path = 2;
    // default false
    optional bool retain_top_folder = 3;
  }

  repeated Repo repos = 1;
  repeated Mirror mirrors = 2;
  repeated Archive archives = 3;

}

// defined enter for tool
message Enter {
  // the priorities are as follows
  // 1. os.arch
  // 2. os
  // 3. *
  map<string, string> shell = 1;
}

```

## Yaml

```yaml
dep:
  plugins:
    - name: java
      file: 
    - name: golang
      repo_file:
        url: xx.git
        ref: xx
        path: plugin.yaml

resource:
  repos:
    - url: "github.com/aa/xx.git"
      ref: master
      path: xx
  mirrros:
    - url: https://xx
      path: xx # download to special path
      executable: false # ensure it is executable if set to true
  archives:
    - url: https://
      path: xx # download and unArchive to special path
      retain_top_folder: false # remove top folder in archiver when un-archive default

enter:
  shell:
      darwin.arm64: echo darwin.arm64
      darwin: echo darwin
      linux: echo "linux"
```

## ProtoText
```prototext
dep: {
  plugins: {
    name: "java"
    file: "dep/java.yaml"
  }
  plugins: {
    name: "golang"
    repo_file: {
      url: "xx.git"
      ref: "xx"
      file: "plugin.yaml"
    }
  }
}
        

resource: {
  repos: {
    url: "github.com/aa/xx.git"
    ref: "master"
    path: "xx"
  }

  mirrros: {
    url: {
      key: "darwin_arm64"
      value: "http://xx"
    }
    url: {
      key: "darwin"
      value: "http://xx"
    }
    path: "xx" # download to special path
    executable: false # ensure it is executable if set to true
  }

  archives: {
    url: {
      key: "darwin_arm64"
      value: "http://xx"
    }
    url: {
      key: "darwin"
      value: "http://xx"
    }
    path: "xx" # download and unArchive to special path
    retain_top_folder: false # remove top folder in archiver when un-archive default
  }
}

enter: {
  shell: {
    key: "darwin_arm64"
    value: "echo darwin.arm64"
  }
  shell: {
    key: "darwin"
    value: "echo darwin"
  }
}
```
