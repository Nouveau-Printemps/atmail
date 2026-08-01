const std = @import("std");
const expect = std.testing.expect;

const SearchResponse = extern struct {
    found: bool = false,
    pos: usize = 0,
    rest: usize = 0,
};

export fn search(haystack: [*:0]const u8, needle: [*:0]const u8) SearchResponse {
    var s = std.mem.span(haystack);
    const n = std.mem.span(needle);
    if (std.simd.suggestVectorLength(u8)) |ln| if (n.len <= ln) {
        const Block = @Vector(ln, u8);
        const sub_first: Block = @splat(n[0]);
        const sub_last: Block = @splat(n[n.len - 1]);
        while (s.len + n.len > ln) : (s = s[ln..]) {
            const first: Block = s[0..][0..ln].*;
            const last: Block = s[n.len - 1 ..][0..ln].*;
            var mask: std.bit_set.IntegerBitSet(ln) = .{
                .mask = @bitCast((sub_first == first) & (sub_last == last)),
            };
            while (mask.toggleFirstSet()) |pos|
                if (std.mem.eql(u8, s[pos..][0..n.len], n))
                    return .{ .found = true, .pos = pos, .rest = pos + n.len };
        }
    };
    const i = std.mem.find(u8, s, n) orelse return .{};
    return .{ .found = true, .pos = i, .rest = n.len + i };
}

test {
    try expect(!search("foo", "bar").found);

    var s = search("foo", "f");
    try expect(s.found);
    try expect(s.pos == 0);
    try expect(s.rest == 1);

    s = search("foo", "o");
    try expect(s.found);
    try expect(s.pos == 1);
    try expect(s.rest == 2);

    const big = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magnam aliquam quaerat voluptatem. Ut enim aeque doleamus animo, cum corpore dolemus, fieri tamen permagna accessio potest, si aliquod aeternum et infinitum impendere malum nobis opinemur. Quod idem licet transferre in voluptatem, ut postea variari.";

    s = search(big, "Lorem");
    try expect(s.found);
    try expect(s.pos == 0);
    try expect(s.rest == 5);

    s = search(big, "consectetur");
    try expect(s.found);
    try expect(s.pos == 28);
    try expect(s.rest == 39);

    s = search(big, "consectetur");
    try expect(s.found);
    try expect(s.pos == 28);
    try expect(s.rest == 39);

    s = search(big, "consectetur adipiscing elit, sed do");
    try expect(s.found);
    try expect(s.pos == 28);
    try expect(s.rest == 63);

    try expect(!search(big, "not_found").found);
    try expect(!search(big, "not_found foo bar hehe, this one must be long").found);
}
