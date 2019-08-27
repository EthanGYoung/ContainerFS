package(licenses = ["notice"])

load("//tools/go_stateify:defs.bzl", "go_library", "go_test")

go_library(
    name = "filter",
    srcs = [
        "filter.go",
        "bloom.go",
    ],
    importpath = "gvisor.googlesource.com/gvisor/pkg/filter",
    visibility = ["//pkg/sentry:internal"],
    deps = [
        "github.com/spaolacci/murmur3",
    ],
)
