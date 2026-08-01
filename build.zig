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

    b.installArtifact(lib);

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
}
