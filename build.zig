const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const mod = b.addModule("search", .{
        .root_source_file = b.path("search/root.zig"),
        .target = target,
        .optimize = optimize,
        .strip = optimize != .Debug,
    });

    const lib = b.addLibrary(.{
        .name = "search",
        .linkage = .static,
        .root_module = mod,
    });
    lib.bundle_compiler_rt = true;
    lib.pie = true;

    const install_lib = b.addInstallArtifact(lib, .{});

    const build_go = buildGo(b, install_lib, "build");
    b.getInstallStep().dependOn(&build_go.step);

    const fmt = b.addFmt(.{
        .paths = &.{
            "search/",
            "build.zig",
            "build.zig.zon",
        },
    });
    lib.step.dependOn(&fmt.step);

    const check = b.step("check", "Check if it compiles");
    check.dependOn(&lib.step);

    const test_step = b.step("test", "Run tests");
    const exe_tests = b.addTest(.{ .root_module = lib.root_module });
    const run_tests = b.addRunArtifact(exe_tests);
    test_step.dependOn(&run_tests.step);

    const run = b.step("run", "Run the application in dev mode");
    const run_go = buildGo(b, install_lib, "run");
    if (b.args) |args| {
        run_go.addArgs(args);
    }
    run.dependOn(&run_go.step);
}

fn buildGo(b: *std.Build, install_lib: *std.Build.Step.InstallArtifact, cmd: []const u8) *std.Build.Step.Run {
    const cc_ldflags = std.fmt.allocPrint(
        b.allocator,
        "{s}/{s}",
        .{ b.lib_dir, install_lib.dest_sub_path },
    ) catch unreachable;

    const build_go = b.addSystemCommand(&[_][]const u8{
        "go",       cmd,
        "-ldflags", "-s -w -linkmode external",
        ".",
    });
    build_go.setEnvironmentVariable("CGO_ENABLED", "1");
    build_go.setEnvironmentVariable("CGO_LDFLAGS", cc_ldflags);
    build_go.step.dependOn(&install_lib.step);
    return build_go;
}
