load("//tools/go_stateify:defs.bzl", "go_library", "go_test")

package(licenses = ["notice"])

go_library(
    name = "compressio",
    srcs = ["compressio.go"],
    importpath = "gvisor.dev/gvisor/pkg/compressio",
    visibility = ["//:sandbox"],
    deps = ["//pkg/binary"],
)

go_test(
    name = "compressio_test",
    size = "medium",
    srcs = ["compressio_test.go"],
    embed = [":compressio"],
)
