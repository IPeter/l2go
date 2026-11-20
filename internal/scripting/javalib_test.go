package scripting_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// javaLib returns the source of scripts/lib/java.lua.
func javaLib(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(scriptsDir, "lib", "java.lua"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// javaPrelude gives a Lua test chunk J and the check helpers. Failures are
// collected; javaEpilogue raises them all at once.
const javaPrelude = `
local J = require("lib.java")
local failures = {}

local function show(v, depth)
  depth = depth or 0
  if type(v) == "string" then
    return string.format("%q", v)
  elseif type(v) ~= "table" or depth > 3 then
    return tostring(v)
  end
  local keys = {}
  for k in pairs(v) do keys[#keys + 1] = k end
  table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
  local parts = {}
  for _, k in ipairs(keys) do
    parts[#parts + 1] = "[" .. show(k) .. "]=" .. show(v[k], depth + 1)
  end
  return "{" .. table.concat(parts, ", ") .. "}"
end

local function eq(a, b)
  if a ~= a and b ~= b then return true end -- NaN
  if type(a) ~= "table" or type(b) ~= "table" then return a == b end
  for k, v in pairs(a) do
    if not eq(v, b[k]) then return false end
  end
  for k in pairs(b) do
    if a[k] == nil then return false end
  end
  return true
end

local function check(name, got, want)
  if not eq(got, want) then
    failures[#failures + 1] = name .. ": got " .. show(got) .. ", want " .. show(want)
  end
end

local function throws(name, fn, part)
  local ok, err = pcall(fn)
  if ok then
    failures[#failures + 1] = name .. ": no error, want one with " .. show(part)
  elseif not string.find(tostring(err), part, 1, true) then
    failures[#failures + 1] = name .. ": error " .. show(tostring(err)) .. ", want one with " .. show(part)
  end
end
`

const javaEpilogue = `
if #failures > 0 then
  error("\n" .. table.concat(failures, "\n"), 0)
end
`

// runJava runs a Lua test chunk with lib/java.lua and fails the test with
// every failed check.
func runJava(t *testing.T, chunk string) {
	t.Helper()
	dir := scripttest.WriteScripts(t, map[string]string{
		"lib/java.lua": javaLib(t),
		"test.lua":     javaPrelude + chunk + javaEpilogue,
	})
	if _, err := scripting.Check(context.Background(), scripting.Options{Dir: dir}); err != nil {
		t.Error(err)
	}
}

// A helper's error names the line of the script that called it, not a
// line of lib/java.lua.
func TestJavaErrorsPointAtCaller(t *testing.T) {
	for _, call := range []struct{ code, msg string }{
		{`local n = J.parse_int("x")`, `NumberFormatException: For input string: "x"`},
		{`local s = J.trim(nil)`, "NullPointerException: trim on null"},
		{`local n = J.idiv(1, 0)`, "ArithmeticException: / by zero"},
		{`J.todo("lambda")`, "not translated: lambda"},
		{`local o = J.obj(5)`, "J.obj needs world.find_object"},
	} {
		dir := scripttest.WriteScripts(t, map[string]string{
			"lib/java.lua": javaLib(t),
			"caller.lua":   "local J = require(\"lib.java\")\n" + call.code + "\n",
		})
		_, err := scripting.Check(context.Background(), scripting.Options{Dir: dir})
		if err == nil {
			t.Errorf("%s: no error", call.code)
			continue
		}
		if want := "caller.lua:2: " + call.msg; !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %q, want it to contain %q", call.code, err, want)
		}
	}
}

func TestJavaIntegerMath(t *testing.T) {
	runJava(t, `
check("idiv 7/2", J.idiv(7, 2), 3)
check("idiv -7/2", J.idiv(-7, 2), -3)
check("idiv 7/-2", J.idiv(7, -2), -3)
check("idiv -7/-2", J.idiv(-7, -2), 3)
check("idiv 0/5", J.idiv(0, 5), 0)
check("idiv -6/3", J.idiv(-6, 3), -2)
check("idiv 1/3", J.idiv(1, 3), 0)
check("idiv -1/3", J.idiv(-1, 3), 0)
check("idiv big", J.idiv(9007199254740991, 3), 3002399751580330)
check("idiv big 2", J.idiv(9007199254740989, 3), 3002399751580329)
check("idiv big 3", J.idiv(-9007199254740991, 2), -4503599627370495)
check("idiv big 4", J.idiv(9007199254740990, 9007199254740991), 0)
check("idiv 32/5", J.idiv(32, 5), 6)
throws("idiv by 0", function() return J.idiv(1, 0) end, "ArithmeticException: / by zero")
throws("idiv 0 by 0", function() return J.idiv(0, 0) end, "ArithmeticException")

check("int 3.7", J.int(3.7), 3)
check("int -3.7", J.int(-3.7), -3)
check("int -0.5", J.int(-0.5), 0)
check("int NaN", J.int(0 / 0), 0)
check("int big", J.int(1e10), 2147483647)
check("int -big", J.int(-1e10), -2147483648)
check("int inf", J.int(1 / 0), 2147483647)
check("int max+0.5", J.int(2147483647.5), 2147483647)
check("int pow", J.int(math.pow(2, 30)), 1073741824)
check("long big", J.int(1e10 + 0.5, "long"), 1e10)
check("long huge", J.int(1e30, "long"), 9223372036854775807)
check("long -huge", J.int(-1e30, "long"), -9223372036854775808)
check("long NaN", J.int(0 / 0, "long"), 0)

check("round 2.5", J.round(2.5), 3)
check("round -2.5", J.round(-2.5), -2)
check("round 2.4", J.round(2.4), 2)
check("round -2.6", J.round(-2.6), -3)
check("round 0.49999999999999994", J.round(0.49999999999999994), 0)
check("round -0.5", J.round(-0.5), 0)
check("round 5", J.round(5), 5)
check("round NaN", J.round(0 / 0), 0)
check("round huge", J.round(1e300), 9223372036854775807)
check("round -huge", J.round(-1e300), -9223372036854775808)

check("constrain in", J.constrain(5, 1, 10), 5)
check("constrain low", J.constrain(0, 1, 10), 1)
check("constrain high", J.constrain(11, 1, 10), 10)
check("constrain min > max", J.constrain(5, 10, 1), 10)
check("constrain double", J.constrain(0.5, 0, 1), 0.5)
`)
}

func TestJavaBits(t *testing.T) {
	runJava(t, `
check("bor", J.bor(1, 2), 3)
check("bor same", J.bor(5, 1), 5)
check("band", J.band(6, 3), 2)
check("bxor", J.bxor(6, 3), 5)
check("band -1", J.band(-1, 255), 255)
check("bor min", J.bor(-2147483648, 1), -2147483647)
check("bxor -1", J.bxor(-1, 0), -1)
check("bxor signs", J.bxor(-1, 1), -2)
check("band negative", J.band(-8, -3), -8)
check("bor wraps 2^31", J.bor(2147483648, 0), -2147483648)
check("bor wraps 2^32", J.bor(4294967296 + 5, 0), 5)
check("bnot 0", J.bnot(0), -1)
check("bnot -1", J.bnot(-1), 0)
check("bnot 5", J.bnot(5), -6)
check("bnot max", J.bnot(2147483647), -2147483648)

check("shl 1<<3", J.shl(1, 3), 8)
check("shl 1<<31", J.shl(1, 31), -2147483648)
check("shl 1<<32", J.shl(1, 32), 1)
check("shl 1<<33", J.shl(1, 33), 2)
check("shl 1<<-1", J.shl(1, -1), -2147483648)
check("shl 3<<30", J.shl(3, 30), -1073741824)
check("shl -1<<4", J.shl(-1, 4), -16)
check("shl overflow", J.shl(1073741824, 1), -2147483648)
check("shl lost", J.shl(-2147483648, 1), 0)
check("shl pack", J.shl(J.shl(5, 8) + 7, 8), 329472)

check("shr 16>>2", J.shr(16, 2), 4)
check("shr -16>>2", J.shr(-16, 2), -4)
check("shr -1>>31", J.shr(-1, 31), -1)
check("shr min>>31", J.shr(-2147483648, 31), -1)
check("shr -7>>1", J.shr(-7, 1), -4)
check("shr 1>>32", J.shr(1, 32), 1)

check("ushr 16>>>2", J.ushr(16, 2), 4)
check("ushr -1>>>28", J.ushr(-1, 28), 15)
check("ushr -1>>>0", J.ushr(-1, 0), -1)
check("ushr -16>>>2", J.ushr(-16, 2), 1073741820)
check("ushr min>>>31", J.ushr(-2147483648, 31), 1)
`)
}

func TestJavaParsing(t *testing.T) {
	runJava(t, `
check("parse_int", J.parse_int("123"), 123)
check("parse_int negative", J.parse_int("-5"), -5)
check("parse_int plus", J.parse_int("+7"), 7)
check("parse_int zeros", J.parse_int("007"), 7)
check("parse_int long", J.parse_int("1700000000000"), 1700000000000)
check("parse_int number", J.parse_int(42), 42)
throws("parse_int empty", function() return J.parse_int("") end, 'NumberFormatException: For input string: ""')
throws("parse_int space", function() return J.parse_int(" 1") end, "NumberFormatException")
throws("parse_int trailing space", function() return J.parse_int("1 ") end, "NumberFormatException")
throws("parse_int decimal", function() return J.parse_int("1.5") end, "NumberFormatException")
throws("parse_int word", function() return J.parse_int("abc") end, 'For input string: "abc"')
throws("parse_int sign only", function() return J.parse_int("-") end, "NumberFormatException")
throws("parse_int exponent", function() return J.parse_int("1e3") end, "NumberFormatException")
throws("parse_int hex", function() return J.parse_int("0x10") end, "NumberFormatException")
throws("parse_int nil", function() return J.parse_int(nil) end, "NumberFormatException: Cannot parse null string")
throws("parse_int too big", function() return J.parse_int("99999999999999999999") end, "NumberFormatException")

check("parse_double", J.parse_double("1.5"), 1.5)
check("parse_double spaces", J.parse_double(" 2 "), 2)
check("parse_double exponent", J.parse_double("-3e2"), -300)
check("parse_double integer", J.parse_double("123"), 123)
check("parse_double exponent without dot", J.parse_double("3e2"), 300)
check("parse_double small exponent", J.parse_double("1.5e-4"), 1.5e-4)
check("parse_double overflow", J.parse_double("1e400"), 1 / 0)
check("parse_double -overflow", J.parse_double("-1e400"), -1 / 0)
check("parse_double underflow", J.parse_double("1e-400"), 0)
check("parse_double zero exponent", J.parse_double("0e400"), 0)
check("parse_double dot first", J.parse_double(".5"), 0.5)
check("parse_double dot last", J.parse_double("5."), 5)
check("parse_double plus", J.parse_double("+1.25E1"), 12.5)
check("parse_double f", J.parse_double("1.5f"), 1.5)
check("parse_double d", J.parse_double("1d"), 1)
check("parse_double NaN", J.parse_double("NaN"), 0 / 0)
check("parse_double -NaN", J.parse_double("-NaN"), 0 / 0)
check("parse_double inf", J.parse_double("Infinity"), 1 / 0)
check("parse_double -inf", J.parse_double("-Infinity"), -1 / 0)
throws("parse_double word", function() return J.parse_double("abc") end, "NumberFormatException")
throws("parse_double empty", function() return J.parse_double("") end, "NumberFormatException")
throws("parse_double hex", function() return J.parse_double("0x10") end, "NumberFormatException")
throws("parse_double two dots", function() return J.parse_double("1.2.3") end, "NumberFormatException")
throws("parse_double lower inf", function() return J.parse_double("infinity") end, "NumberFormatException")
throws("parse_double nil", function() return J.parse_double(nil) end, "NullPointerException")

check("parse_bool true", J.parse_bool("true"), true)
check("parse_bool TRUE", J.parse_bool("TrUe"), true)
check("parse_bool yes", J.parse_bool("yes"), false)
check("parse_bool space", J.parse_bool("true "), false)
check("parse_bool nil", J.parse_bool(nil), false)

check("is_numeric", J.is_numeric("123"), true)
check("is_numeric 0", J.is_numeric("0"), true)
check("is_numeric empty", J.is_numeric(""), false)
check("is_numeric nil", J.is_numeric(nil), false)
check("is_numeric negative", J.is_numeric("-1"), false)
check("is_numeric decimal", J.is_numeric("1.5"), false)
check("is_numeric mixed", J.is_numeric("12a"), false)
check("is_numeric space", J.is_numeric(" 1"), false)
`)
}

func TestJavaDoubleToString(t *testing.T) {
	runJava(t, `
check("str 1", J.str(1), "1.0")
check("str 0.25", J.str(0.25), "0.25")
check("str -3.5", J.str(-3.5), "-3.5")
check("str 100", J.str(100), "100.0")
check("str 0.1", J.str(0.1), "0.1")
check("str 0.3", J.str(0.3), "0.3")
check("str 1/3", J.str(1 / 3), "0.3333333333333333")
check("str 2/3", J.str(2 / 3), "0.6666666666666666")
check("str 0.1+0.2", J.str(0.1 + 0.2), "0.30000000000000004")
check("str 0.001", J.str(0.001), "0.001")
check("str 9999999", J.str(9999999), "9999999.0")
check("str 1e7", J.str(1e7), "1.0E7")
check("str 12345678.9", J.str(12345678.9), "1.23456789E7")
check("str 1.5e-4", J.str(1.5e-4), "1.5E-4")
check("str 1e-5", J.str(1e-5), "1.0E-5")
check("str -2e-7", J.str(-2e-7), "-2.0E-7")
check("str 1e21", J.str(1e21), "1.0E21")
check("str 2^53", J.str(2 ^ 53), "9.007199254740992E15")
check("str NaN", J.str(0 / 0), "NaN")
check("str inf", J.str(1 / 0), "Infinity")
check("str -inf", J.str(-1 / 0), "-Infinity")
check("str 0", J.str(0), "0.0")
check("str nil", J.str(nil), "null")
check("str string", J.str("x"), "x")
`)
}

func TestJavaStrings(t *testing.T) {
	runJava(t, `
check("equals_ignore_case", J.equals_ignore_case("Hello", "hELLO"), true)
check("equals_ignore_case differ", J.equals_ignore_case("a", "b"), false)
check("equals_ignore_case length", J.equals_ignore_case("a", "aa"), false)
check("equals_ignore_case nil", J.equals_ignore_case("a", nil), false)
check("equals_ignore_case number", J.equals_ignore_case("1", 1), false)
throws("equals_ignore_case on nil", function() return J.equals_ignore_case(nil, "a") end, "NullPointerException")

check("starts_with", J.starts_with("quest_accept", "quest"), true)
check("starts_with longer", J.starts_with("ab", "abc"), false)
check("starts_with empty", J.starts_with("ab", ""), true)
check("starts_with pattern chars", J.starts_with("a.b", "."), false)
throws("starts_with nil", function() return J.starts_with("a", nil) end, "NullPointerException")
check("ends_with", J.ends_with("file.htm", ".htm"), true)
check("ends_with longer", J.ends_with("htm", "x.htm"), false)
check("ends_with empty", J.ends_with("a", ""), true)
check("ends_with no", J.ends_with("file.html", ".htm"), false)

check("str_contains", J.str_contains("a.b", "."), true)
check("str_contains plain", J.str_contains("abc", "%a"), false)
check("str_contains empty", J.str_contains("abc", ""), true)
check("str_contains missing", J.str_contains("abc", "d"), false)

check("substring from", J.substring("hello", 1), "ello")
check("substring range", J.substring("hello", 1, 3), "el")
check("substring end", J.substring("hello", 5), "")
check("substring empty", J.substring("hello", 0, 0), "")
check("substring all", J.substring("hello", 0, 5), "hello")
throws("substring negative", function() return J.substring("hello", -1) end, "StringIndexOutOfBoundsException")
throws("substring past end", function() return J.substring("hello", 2, 6) end, "StringIndexOutOfBoundsException: begin 2, end 6, length 5")
throws("substring reversed", function() return J.substring("hello", 3, 2) end, "StringIndexOutOfBoundsException")
throws("substring begin past end", function() return J.substring("hello", 6) end, "StringIndexOutOfBoundsException")

check("index_of", J.index_of("hello", "l"), 2)
check("index_of from", J.index_of("hello", "l", 3), 3)
check("index_of missing", J.index_of("hello", "z"), -1)
check("index_of empty", J.index_of("hello", ""), 0)
check("index_of empty past end", J.index_of("hello", "", 10), 5)
check("index_of negative from", J.index_of("hello", "l", -5), 2)
check("index_of from at end", J.index_of("hello", "o", 5), -1)
check("index_of plain", J.index_of("a.b", "."), 1)
check("index_of word", J.index_of("1_2_3", "_2"), 1)

check("last_index_of", J.last_index_of("hello", "l"), 3)
check("last_index_of overlap", J.last_index_of("aaa", "aa"), 1)
check("last_index_of missing", J.last_index_of("abc", "z"), -1)
check("last_index_of empty", J.last_index_of("abc", ""), 3)
check("last_index_of plain", J.last_index_of("a.b.c", "."), 3)

check("char_at 0", J.char_at("abc", 0), "a")
check("char_at 2", J.char_at("abc", 2), "c")
throws("char_at past end", function() return J.char_at("abc", 3) end, "StringIndexOutOfBoundsException")
throws("char_at negative", function() return J.char_at("abc", -1) end, "StringIndexOutOfBoundsException")

check("trim", J.trim("  a b \t\n"), "a b")
check("trim empty", J.trim(""), "")
check("trim blank", J.trim("   "), "")
check("trim control", J.trim("\0x\1"), "x")
check("trim none", J.trim("x"), "x")
throws("trim nil", function() return J.trim(nil) end, "NullPointerException")

check("replace", J.replace("a.b.c", ".", "-"), "a-b-c")
check("replace placeholder", J.replace("%name%, hi %name%!", "%name%", "Bob"), "Bob, hi Bob!")
check("replace left to right", J.replace("aaa", "aa", "b"), "ba")
check("replace empty target", J.replace("ab", "", "-"), "-a-b-")
check("replace missing", J.replace("abc", "x", "y"), "abc")
check("replace percent", J.replace("a%b", "%", "%%"), "a%%b")
check("replace literal replacement", J.replace("x", "x", "%1$0"), "%1$0")
check("replace to empty", J.replace("a-b", "-", ""), "ab")

check("replace_first", J.replace_first("a.b.a", "a", "X"), "X.b.a")
check("replace_first missing", J.replace_first("abc", "z", "y"), "abc")
check("replace_first escape", J.replace_first("abc", "b", "\\$"), "a$c")
check("replace_first group 0", J.replace_first("abc", "b", "[$0]"), "a[b]c")
throws("replace_first group 1", function() return J.replace_first("abc", "b", "$1") end, "IndexOutOfBoundsException: No group 1")
throws("replace_first lone dollar", function() return J.replace_first("abc", "b", "$") end, "IllegalArgumentException")

check("split", J.split("a,b,c", ","), {"a", "b", "c"})
check("split inner empty", J.split("a,,b", ","), {"a", "", "b"})
check("split trailing empty", J.split("a,b,,", ","), {"a", "b"})
check("split leading empty", J.split(",a", ","), {"", "a"})
check("split empty string", J.split("", ","), {""})
check("split only separators", J.split(",,", ","), {})
check("split no separator", J.split("abc", ","), {"abc"})
check("split dot", J.split("a.b", "."), {"a", "b"})
check("split chars", J.split("abc", ""), {"a", "b", "c"})
check("split long separator", J.split("a||b", "||"), {"a", "b"})
check("split space", J.split("1 2 3", " "), {"1", "2", "3"})
`)
}

func TestJavaCollections(t *testing.T) {
	runJava(t, `
check("array", J.array(0, 3), {0, 0, 0})
check("array bool", J.array(false, 2), {false, false})
check("array 2d", J.array(0, 2, 3), {{0, 0, 0}, {0, 0, 0}})
check("array empty", J.array(0, 0), {})
local a2 = J.array(0, 2, 2)
a2[1][1] = 5
check("array rows are distinct", a2[2][1], 0)
throws("array negative", function() return J.array(0, -1) end, "NegativeArraySizeException")

local orig = {1, 2, 3}
local c = J.copy(orig)
check("copy", c, {1, 2, 3})
c[1] = 9
check("copy is shallow", orig[1], 1)
check("copy map", J.copy({a = 1, b = 2}), {a = 1, b = 2})
throws("copy nil", function() return J.copy(nil) end, "NullPointerException")

check("copy_of shorter", J.copy_of({1, 2, 3}, 2), {1, 2})
check("copy_of longer", J.copy_of({1, 2}, 4), {1, 2, 0, 0})
check("copy_of strings", J.copy_of({"a"}, 2), {"a"})
check("copy_of bools", J.copy_of({true}, 2), {true, false})
check("copy_of default", J.copy_of({}, 2, 0), {0, 0})
check("copy_of zero", J.copy_of({1, 2}, 0), {})
throws("copy_of negative", function() return J.copy_of({1}, -1) end, "NegativeArraySizeException")

local filled = {1, 2, 3}
J.fill(filled, 7)
check("fill", filled, {7, 7, 7})

rnd = function(n) return 0 end
local s1 = {1, 2, 3}
J.shuffle(s1)
check("shuffle first", s1, {2, 3, 1})
rnd = function(n) return n - 1 end
local s2 = {1, 2, 3}
J.shuffle(s2)
check("shuffle identity", s2, {1, 2, 3})
rnd = nil
local s3 = {1, 2, 3, 4, 5, 6, 7, 8}
J.shuffle(s3)
table.sort(s3)
check("shuffle permutation", s3, {1, 2, 3, 4, 5, 6, 7, 8})
local s4 = {}
J.shuffle(s4)
check("shuffle empty", s4, {})

check("contains", J.contains({1, 2, 3}, 2), true)
check("contains missing", J.contains({1, 2}, 5), false)
check("contains string", J.contains({"a", "b"}, "b"), true)
check("contains nil", J.contains({}, nil), false)

check("index_of_value", J.index_of_value({5, 6, 5}, 5), 0)
check("index_of_value second", J.index_of_value({5, 6}, 6), 1)
check("index_of_value missing", J.index_of_value({5}, 7), -1)

local l = {}
check("add", J.add(l, 4), true)
check("add list", l, {4})

local m = {1}
check("add_all", J.add_all(m, {2, 3}), true)
check("add_all list", m, {1, 2, 3})
check("add_all nothing", J.add_all(m, {}), false)
check("add_all set", J.add_all(m, {[9] = true, [8] = true}), true)
check("add_all set list", m, {1, 2, 3, 8, 9})

local r = {1, 2, 1}
check("remove_value", J.remove_value(r, 1), true)
check("remove_value list", r, {2, 1})
check("remove_value missing", J.remove_value(r, 5), false)

check("list_to_string", J.list_to_string({1, 2, 3}), "[1, 2, 3]")
check("list_to_string empty", J.list_to_string({}), "[]")
check("list_to_string mixed", J.list_to_string({"a", 1.5, true}), "[a, 1.5, true]")
check("list_to_string nested", J.list_to_string({{1}, {2}}), "[[1], [2]]")
check("list_to_string negative", J.list_to_string({-3, 0}), "[-3, 0]")
check("list_to_string big", J.list_to_string({1700000000000}), "[1700000000000]")
`)
}

func TestJavaMaps(t *testing.T) {
	runJava(t, `
check("keys sorted", J.keys({b = 1, a = 2, [3] = 1, [1] = 1}), {1, 3, "a", "b"})
check("keys set", J.keys({[30] = true, [10] = true, [20] = true}), {10, 20, 30})
check("keys empty", J.keys({}), {})
check("values", J.values({b = "x", a = "y", c = "z"}), {"y", "x", "z"})

local order = {}
for k, v in J.spairs({c = 3, a = 1, b = 2}) do
  order[#order + 1] = k .. "=" .. v
end
check("spairs", order, {"a=1", "b=2", "c=3"})

local t = {a = 1, b = 2, c = 3}
local seen = {}
for k in J.spairs(t) do
  if k == "a" then t.b = nil end
  seen[#seen + 1] = k
end
check("spairs skips removed", seen, {"a", "c"})

local n = 0
for _ in J.spairs({}) do n = n + 1 end
check("spairs empty", n, 0)

check("size empty", J.size({}), 0)
check("size", J.size({a = 1, [1] = 2, [5] = 3}), 3)
throws("size nil", function() return J.size(nil) end, "NullPointerException")

local cl = {1, 2, a = 3}
J.clear(cl)
check("clear", cl, {})

local m = {}
check("put new", J.put(m, "a", 1), nil)
check("put old", J.put(m, "a", 2), 1)
check("put map", m, {a = 2})

local pa = {a = 1}
check("put_if_absent present", J.put_if_absent(pa, "a", 5), 1)
check("put_if_absent absent", J.put_if_absent(pa, "b", 5), nil)
check("put_if_absent map", pa, {a = 1, b = 5})

local rk = {a = 1, b = 2}
check("remove_key", J.remove_key(rk, "a"), 1)
check("remove_key missing", J.remove_key(rk, "zz"), nil)
check("remove_key map", rk, {b = 2})

check("get_or_default present", J.get_or_default({a = 1}, "a", 9), 1)
check("get_or_default absent", J.get_or_default({}, "a", 9), 9)
check("get_or_default false", J.get_or_default({a = false}, "a", true), false)

check("contains_value", J.contains_value({a = 1, b = 2}, 2), true)
check("contains_value missing", J.contains_value({a = 1}, 3), false)

local s = J.set_of({1, 2, 2})
check("set_of", s, {[1] = true, [2] = true})
check("set_add new", J.set_add(s, 3), true)
check("set_add again", J.set_add(s, 3), false)
check("set_remove", J.set_remove(s, 3), true)
check("set_remove again", J.set_remove(s, 3), false)
check("set after", s, {[1] = true, [2] = true})
`)
}

func TestJavaMarkersAndRefs(t *testing.T) {
	runJava(t, `
throws("todo", function() J.todo("lambda") end, "not translated: lambda")
throws("todo with java", function() J.todo("lambda", "x -> y") end, "not translated: lambda (x -> y)")
check("oid nil", J.oid(nil), nil)
check("oid number", J.oid(7), 7)
check("obj nil", J.obj(nil), nil)
throws("obj without world", function() return J.obj(1) end, "world.find_object")
check("instance_of nil", J.instance_of(nil, "Npc"), false)
check("instance_of string", J.instance_of("x", "String"), true)
check("instance_of number", J.instance_of(1, "String"), false)
`)

	// Player and Npc values exist only inside handlers.
	h := scripttest.LoadSource(t, map[string]string{
		"lib/java.lua": javaLib(t),
		"refs.lua": `
local J = require("lib.java")
local objects = {}
world = { find_object = function(id) return objects[id] end }

local s = script.new("JavaRefs")
s:on_event(function(event, npc, player)
  local out = {}
  local function add(v) out[#out + 1] = tostring(v) end
  if event == "refs" then
    objects[npc:object_id()] = npc
    local kept = J.oid(npc)
    add(kept)
    add(J.obj(kept) == npc)
    add(J.obj(12345))
    add(J.oid(player))
  elseif event == "types" then
    add(J.instance_of(player, "Player"))
    add(J.instance_of(player, "Creature"))
    add(J.instance_of(player, "Npc"))
    add(J.instance_of(npc, "Npc"))
    add(J.instance_of(npc, "WorldObject"))
    add(J.instance_of(npc, "Player"))
    local ok, err = pcall(J.instance_of, npc, "Attackable")
    add(ok)
    add(string.find(tostring(err), "npc:is_type", 1, true) ~= nil)
  end
  return table.concat(out, ",")
end)
`,
	})
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 20}
	guard := &scripttest.Npc{ObjectID: 0x60000001, ID: 30733, Name: "Guard"}

	reply, _ := h.Event("JavaRefs", "refs", hero, guard)
	if want := "1610612737,true,nil,268435457"; reply.Content != want {
		t.Errorf("refs = %q, want %q", reply.Content, want)
	}
	reply, _ = h.Event("JavaRefs", "types", hero, guard)
	if want := strings.Join([]string{"true", "true", "false", "true", "true", "false", "false", "true"}, ","); reply.Content != want {
		t.Errorf("types = %q, want %q", reply.Content, want)
	}
}
