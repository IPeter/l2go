-- Java library behaviour for quests translated from Mobius (cmd/questport).
--
--   local J = require("lib.java")
--
-- Each helper does what the Java method it stands for does, with these
-- conventions:
--   * Indexes are 0-based where Java's are (substring, index_of, char_at),
--     and results that are indexes stay 0-based (-1 when not found).
--   * Lists and arrays are 1-based Lua sequences, maps are Lua tables and
--     sets are tables of value = true. Java's null is nil.
--   * Numbers are Lua numbers (float64, exact to 2^53). Integer helpers
--     follow Java's int rules: division truncates toward zero, casts clamp,
--     bit operations work on 32-bit two's complement values.
--   * Where Java throws, the helper raises a Lua error whose message starts
--     with the Java exception name, e.g. "NumberFormatException: ...".
--   * Strings are byte strings: lengths and indexes count bytes, which are
--     Java chars for the ASCII text quests handle.

---@type JavaLib
local J = {}

local floor, ceil = math.floor, math.ceil
local huge = 1 / 0 -- math.huge is the largest finite number in gopher-lua
local sfind, ssub, sbyte, slower, sformat = string.find, string.sub, string.byte, string.lower, string.format
local tremove, tsort, tconcat = table.remove, table.sort, table.concat

local TWO32 = 4294967296
local TWO31 = 2147483648
local INT_MIN, INT_MAX = -2147483648, 2147483647
local LONG_MIN, LONG_MAX = -9223372036854775808, 9223372036854775807

-- Error levels: gopher-lua counts one more frame than PUC Lua, so level 3
-- is the caller of the function that calls error, and level 4 the caller
-- of a J helper that calls throw or need_*.
local CALLER, HELPER_CALLER = 3, 4

-- throw raises a Java-style exception at the caller of the helper.
local function throw(name, msg)
  error(name .. ": " .. msg, HELPER_CALLER)
end

local function need_string(s, what)
  if type(s) ~= "string" then
    if s == nil then
      error("NullPointerException: " .. what .. " on null", HELPER_CALLER)
    end
    error("ClassCastException: " .. what .. " needs a string, got " .. type(s), HELPER_CALLER)
  end
end

local function need_table(t, what)
  if type(t) ~= "table" then
    if t == nil then
      error("NullPointerException: " .. what .. " on null", HELPER_CALLER)
    end
    error("ClassCastException: " .. what .. " needs a table, got " .. type(t), HELPER_CALLER)
  end
end

local function is_integer(x)
  return type(x) == "number" and x == floor(x) and x > -huge and x < huge
end

-------------------------------------------------------------------------------
-- Translator markers
-------------------------------------------------------------------------------

-- J.todo(reason[, java]) stands for Java code the translator could not
-- translate. The script loads; the call fails when it is reached.
function J.todo(reason, java)
  local msg = "not translated: " .. tostring(reason)
  if java ~= nil then
    msg = msg .. " (" .. tostring(java) .. ")"
  end
  error(msg, CALLER)
end

-------------------------------------------------------------------------------
-- Integer arithmetic
-------------------------------------------------------------------------------

-- J.idiv(a, b) is Java's integer a / b: truncated toward zero, and an
-- ArithmeticException when b is 0. Exact for operands up to 2^53.
function J.idiv(a, b)
  if b == 0 then
    throw("ArithmeticException", "/ by zero")
  end
  if not (is_integer(a) and is_integer(b)) then
    local q = a / b
    if q ~= q then
      return 0
    end
    return q >= 0 and floor(q) or ceil(q)
  end
  local A, B = a < 0 and -a or a, b < 0 and -b or b
  local q = floor(A / B)
  -- The float division may round across an integer for big operands.
  local r = A - q * B
  if r < 0 then
    q = q - 1
  elseif r >= B then
    q = q + 1
  end
  if (a < 0) ~= (b < 0) then
    q = -q
  end
  return q
end

-- J.int(x[, kind]) is Java's (int) cast of a double: truncated toward zero,
-- NaN is 0, and values outside the int range clamp to Integer.MIN_VALUE or
-- MAX_VALUE. With kind "long" it is the (long) cast and clamps to the long
-- range instead.
function J.int(x, kind)
  if x ~= x then
    return 0
  end
  local lo, hi = INT_MIN, INT_MAX
  if kind == "long" then
    lo, hi = LONG_MIN, LONG_MAX
  end
  if x <= lo then
    return lo
  end
  if x >= hi then
    return hi
  end
  return x >= 0 and floor(x) or ceil(x)
end

-- J.round(x) is Math.round(double): the closest long, ties toward positive
-- infinity (2.5 is 3, -2.5 is -2); NaN is 0; clamps to the long range.
function J.round(x)
  if x ~= x then
    return 0
  end
  if x <= LONG_MIN then
    return LONG_MIN
  end
  if x >= LONG_MAX then
    return LONG_MAX
  end
  local f = floor(x)
  if x - f >= 0.5 then
    return f + 1
  end
  return f
end

-- J.constrain(value, min, max) is Mobius MathUtil.clamp (and the older
-- CommonUtil.constrain): max(min, min(max, value)).
function J.constrain(value, min, max)
  return math.max(min, math.min(max, value))
end

-------------------------------------------------------------------------------
-- Bit operations on 32-bit ints
-------------------------------------------------------------------------------

-- u32 narrows x to an int, as unsigned 0 .. 2^32-1.
local function u32(x)
  if x ~= x or x == huge or x == -huge then
    return 0
  end
  x = x >= 0 and floor(x) or ceil(x)
  return x - floor(x / TWO32) * TWO32
end

-- s32 turns an unsigned 32-bit value into a signed int.
local function s32(u)
  if u >= TWO31 then
    return u - TWO32
  end
  return u
end

local function bitop(a, b, keep)
  a, b = u32(a), u32(b)
  local r, bit = 0, 1
  for _ = 1, 32 do
    local x, y = a % 2, b % 2
    if keep(x, y) then
      r = r + bit
    end
    a, b, bit = (a - x) / 2, (b - y) / 2, bit * 2
  end
  return s32(r)
end

local function op_and(x, y) return x == 1 and y == 1 end
local function op_or(x, y) return x == 1 or y == 1 end
local function op_xor(x, y) return x ~= y end

-- J.band(a, b) is Java's int a & b.
function J.band(a, b) return bitop(a, b, op_and) end

-- J.bor(a, b) is Java's int a | b.
function J.bor(a, b) return bitop(a, b, op_or) end

-- J.bxor(a, b) is Java's int a ^ b.
function J.bxor(a, b) return bitop(a, b, op_xor) end

-- J.bnot(a) is Java's int ~a.
function J.bnot(a) return s32(TWO32 - 1 - u32(a)) end

-- shift returns the shift distance as Java uses it: the low 5 bits.
local function shift(n)
  return u32(n) % 32
end

-- J.shl(a, n) is Java's int a << n (bits shifted out are lost, the sign
-- bit can be set).
function J.shl(a, n)
  n = shift(n)
  local p = 2 ^ n
  local low = u32(a) % (TWO32 / p) -- the bits that stay
  return s32(low * p)
end

-- J.shr(a, n) is Java's int a >> n (sign-extending).
function J.shr(a, n)
  return floor(s32(u32(a)) / 2 ^ shift(n))
end

-- J.ushr(a, n) is Java's int a >>> n (zero-filling).
function J.ushr(a, n)
  return s32(floor(u32(a) / 2 ^ shift(n)))
end

-------------------------------------------------------------------------------
-- Number parsing and printing
-------------------------------------------------------------------------------

-- parse_integral parses an optional sign and ASCII digits within lo .. hi
-- for parse_int and parse_long. It returns the number, or nil and the
-- NumberFormatException message (the helpers throw, so the error points
-- at their caller).
local function parse_integral(s, lo, hi)
  if s == nil then
    return nil, "Cannot parse null string: null"
  end
  if type(s) == "number" then
    s = tostring(s)
  end
  if type(s) ~= "string" or not sfind(s, "^[+-]?%d+$") then
    return nil, 'For input string: "' .. tostring(s) .. '"'
  end
  local n = tonumber(ssub(s, 1, 1) == "+" and ssub(s, 2) or s)
  if n == nil or n < lo or n > hi then
    return nil, 'For input string: "' .. s .. '"'
  end
  return n
end

-- J.parse_int(s) is Integer.parseInt: an optional sign and ASCII digits,
-- nothing else (no spaces), within the int range. Anything else raises a
-- NumberFormatException.
function J.parse_int(s)
  local n, err = parse_integral(s, INT_MIN, INT_MAX)
  if n == nil then
    throw("NumberFormatException", err)
  end
  return n
end

-- J.parse_long(s) is Long.parseLong: as parse_int, within the long range.
-- Values beyond 2^53 lose precision.
function J.parse_long(s)
  local n, err = parse_integral(s, LONG_MIN, LONG_MAX)
  if n == nil then
    throw("NumberFormatException", err)
  end
  return n
end

-- trim_bounds returns the first and last byte that Java's trim keeps.
local function trim_bounds(s)
  local i, j = 1, #s
  while i <= j and sbyte(s, i) <= 32 do
    i = i + 1
  end
  while j >= i and sbyte(s, j) <= 32 do
    j = j - 1
  end
  return i, j
end

-- decimal reads a checked decimal number: sign, digits with an optional
-- dot, optional exponent. gopher-lua's tonumber refuses an exponent after a
-- mantissa without a dot ("3e2") and values out of the double range, so the
-- mantissa gets a dot and a range error becomes 0 or infinity, as in Java.
local function decimal(body)
  local sign, mant, exp = body:match("^([+-]?)([%d%.]+)[eE]?([+-]?%d*)$")
  if not sfind(mant, ".", 1, true) then
    mant = mant .. ".0"
  end
  if ssub(mant, 1, 1) == "." then
    mant = "0" .. mant
  end
  if ssub(mant, -1) == "." then
    mant = mant .. "0"
  end
  local text = mant
  if exp ~= "" then
    text = text .. "e" .. exp
  end
  local n = tonumber(text)
  if n == nil then
    if not sfind(mant, "[1-9]") or ssub(exp, 1, 1) == "-" then
      n = 0 -- underflow
    else
      n = huge -- overflow
    end
  end
  if sign == "-" then
    n = -n
  end
  return n
end

-- J.parse_double(s) is Double.parseDouble: surrounding whitespace is
-- ignored; a decimal number with optional exponent and an optional f/F/d/D
-- suffix, "NaN" or "Infinity" (with an optional sign). Hexadecimal
-- floating-point literals are not supported. Anything else raises a
-- NumberFormatException.
function J.parse_double(s)
  if s == nil then
    throw("NullPointerException", "parse_double on null")
  end
  local t = s
  if type(t) == "string" then
    local i, j = trim_bounds(t)
    t = ssub(t, i, j)
    local sign, word = t:match("^([+-]?)(%a+)$")
    if word == "NaN" then
      return 0 / 0
    elseif word == "Infinity" then
      return sign == "-" and -huge or huge
    end
    local body = t:match("^([+-]?%d+%.?%d*[eE][+-]?%d+)[fFdD]?$")
      or t:match("^([+-]?%.%d+[eE][+-]?%d+)[fFdD]?$")
      or t:match("^([+-]?%d+%.?%d*)[fFdD]?$")
      or t:match("^([+-]?%.%d+)[fFdD]?$")
    if body then
      return decimal(body)
    end
  end
  throw("NumberFormatException", 'For input string: "' .. tostring(s) .. '"')
end

-- J.parse_bool(s) is Boolean.parseBoolean: true only for "true" in any
-- case; nil and everything else is false.
function J.parse_bool(s)
  return type(s) == "string" and slower(s) == "true"
end

-- J.is_numeric(s) is Mobius StringUtil.isNumeric (and isDigit): a non-empty
-- string of digits only, so "-1" and "1.5" are not numeric.
function J.is_numeric(s)
  return type(s) == "string" and sfind(s, "^%d+$") ~= nil
end

-- integral_string prints an integral number without a fraction.
local function integral_string(x)
  local s = sformat("%.0f", x)
  if s == "-0" then
    return "0"
  end
  return s
end

-- J.str(x) is String.valueOf(double): Java prints doubles with at least
-- one fraction digit ("1.0", "0.25"), in scientific notation below 10^-3
-- and from 10^7 on ("1.0E7", "1.5E-4"), with the fewest digits that read
-- back as the same double. NaN, Infinity and -0.0 as Java writes them.
-- Non-numbers are printed with tostring (nil is "null").
function J.str(x)
  if type(x) ~= "number" then
    if x == nil then
      return "null"
    end
    return tostring(x)
  end
  if x ~= x then
    return "NaN"
  elseif x == huge then
    return "Infinity"
  elseif x == -huge then
    return "-Infinity"
  elseif x == 0 then
    return 1 / x < 0 and "-0.0" or "0.0"
  end
  local a = x < 0 and -x or x
  if a >= 1e-3 and a < 1e7 then
    local s
    for d = 1, 25 do
      s = sformat("%." .. d .. "f", x)
      if tonumber(s) == x then
        break
      end
    end
    -- Drop zeros the shortest form does not need, keeping one digit.
    s = s:gsub("0+$", "")
    if ssub(s, -1) == "." then
      s = s .. "0"
    end
    return s
  end
  local s
  for p = 0, 17 do
    s = sformat("%." .. p .. "e", x)
    if tonumber(s) == x then
      break
    end
  end
  local mant, exp = s:match("^(-?[%d%.]+)[eE]([-+]?%d+)$")
  if not sfind(mant, ".", 1, true) then
    mant = mant .. ".0"
  else
    mant = mant:gsub("0+$", "")
    if ssub(mant, -1) == "." then
      mant = mant .. "0"
    end
  end
  return mant .. "E" .. tostring(tonumber(exp))
end

-------------------------------------------------------------------------------
-- Strings
-------------------------------------------------------------------------------

-- J.equals_ignore_case(s, other) is String.equalsIgnoreCase: false when
-- other is nil or not a string.
function J.equals_ignore_case(s, other)
  need_string(s, "equalsIgnoreCase")
  if type(other) ~= "string" then
    return false
  end
  return #s == #other and slower(s) == slower(other)
end

-- J.starts_with(s, prefix) is String.startsWith.
function J.starts_with(s, prefix)
  need_string(s, "startsWith")
  need_string(prefix, "startsWith")
  return ssub(s, 1, #prefix) == prefix
end

-- J.ends_with(s, suffix) is String.endsWith.
function J.ends_with(s, suffix)
  need_string(s, "endsWith")
  need_string(suffix, "endsWith")
  return suffix == "" or ssub(s, -#suffix) == suffix
end

-- J.str_contains(s, part) is String.contains: a plain substring search.
function J.str_contains(s, part)
  need_string(s, "contains")
  need_string(part, "contains")
  return sfind(s, part, 1, true) ~= nil
end

-- J.substring(s, begin[, end]) is String.substring with Java's 0-based,
-- end-exclusive indexes. Bad indexes raise a
-- StringIndexOutOfBoundsException.
function J.substring(s, b, e)
  need_string(s, "substring")
  local n = #s
  if e == nil then
    e = n
  end
  if b < 0 or e > n or b > e then
    throw("StringIndexOutOfBoundsException", "begin " .. b .. ", end " .. e .. ", length " .. n)
  end
  return ssub(s, b + 1, e)
end

-- J.index_of(s, part[, from]) is String.indexOf: the 0-based index of the
-- first occurrence at or after from, or -1.
function J.index_of(s, part, from)
  need_string(s, "indexOf")
  need_string(part, "indexOf")
  local n = #s
  from = from or 0
  if from < 0 then
    from = 0
  end
  if from >= n then
    return part == "" and n or -1
  end
  local i = sfind(s, part, from + 1, true)
  return i and i - 1 or -1
end

-- J.last_index_of(s, part) is String.lastIndexOf: the 0-based index of the
-- last occurrence, or -1. An empty part is found at the end.
function J.last_index_of(s, part)
  need_string(s, "lastIndexOf")
  need_string(part, "lastIndexOf")
  if part == "" then
    return #s
  end
  local last, init = -1, 1
  while true do
    local i = sfind(s, part, init, true)
    if not i then
      return last
    end
    last, init = i - 1, i + 1
  end
end

-- J.char_at(s, i) is String.charAt with a 0-based index; the char is a
-- one-byte string.
function J.char_at(s, i)
  need_string(s, "charAt")
  if i < 0 or i >= #s then
    throw("StringIndexOutOfBoundsException", "Index " .. i .. " out of bounds for length " .. #s)
  end
  return ssub(s, i + 1, i + 1)
end

-- J.trim(s) is String.trim: removes bytes up to and including the space
-- (control characters too) from both ends.
function J.trim(s)
  need_string(s, "trim")
  return ssub(s, trim_bounds(s))
end

-- J.replace(s, target, replacement) is String.replace: every occurrence of
-- the literal target, left to right. An empty target inserts the
-- replacement before every char and at the end, as Java does.
function J.replace(s, target, replacement)
  need_string(s, "replace")
  need_string(target, "replace")
  need_string(replacement, "replace")
  local parts = {}
  if target == "" then
    for i = 1, #s do
      parts[#parts + 1] = replacement
      parts[#parts + 1] = ssub(s, i, i)
    end
    parts[#parts + 1] = replacement
    return tconcat(parts)
  end
  local init = 1
  while true do
    local i, j = sfind(s, target, init, true)
    if not i then
      break
    end
    parts[#parts + 1] = ssub(s, init, i - 1)
    parts[#parts + 1] = replacement
    init = j + 1
  end
  if init == 1 then
    return s
  end
  parts[#parts + 1] = ssub(s, init)
  return tconcat(parts)
end

-- regex_replacement expands a Java regex replacement string for a match of
-- a pattern without groups: \x is x, $0 is the match, other groups do not
-- exist. It is called by J helpers, so its errors point one frame further
-- up than throw's.
local function regex_replacement(rep, match)
  if not sfind(rep, "[\\$]") then
    return rep
  end
  local function fail(name, msg)
    error(name .. ": " .. msg, HELPER_CALLER + 1)
  end
  local out, i, n = {}, 1, #rep
  while i <= n do
    local c = ssub(rep, i, i)
    if c == "\\" then
      if i == n then
        fail("IllegalArgumentException", "character to be escaped is missing")
      end
      out[#out + 1] = ssub(rep, i + 1, i + 1)
      i = i + 2
    elseif c == "$" then
      local digits = rep:match("^%d+", i + 1)
      if not digits then
        fail("IllegalArgumentException", "Illegal group reference")
      end
      if tonumber(digits) ~= 0 then
        fail("IndexOutOfBoundsException", "No group " .. digits)
      end
      out[#out + 1] = match
      i = i + 1 + #digits
    else
      out[#out + 1] = c
      i = i + 1
    end
  end
  return tconcat(out)
end

-- J.replace_first(s, target, replacement) is String.replaceFirst with a
-- pattern made of literal characters only: the first occurrence of target
-- is replaced. The replacement follows Java's rules: \x stands for x and
-- $0 for the match.
function J.replace_first(s, target, replacement)
  need_string(s, "replaceFirst")
  need_string(target, "replaceFirst")
  need_string(replacement, "replaceFirst")
  local i, j = sfind(s, target, 1, true)
  if not i then
    return s
  end
  return ssub(s, 1, i - 1) .. regex_replacement(replacement, target) .. ssub(s, j + 1)
end

-- J.replace_all(s, target, replacement) is String.replaceAll with a
-- pattern made of literal characters only: every occurrence of target is
-- replaced, the replacement following Java's rules as in replace_first.
-- Without a match the replacement is not checked, as in Java.
function J.replace_all(s, target, replacement)
  need_string(s, "replaceAll")
  need_string(target, "replaceAll")
  need_string(replacement, "replaceAll")
  if target ~= "" and not sfind(s, target, 1, true) then
    return s
  end
  return J.replace(s, target, regex_replacement(replacement, target))
end

-- J.split(s, sep) is String.split with a literal separator: trailing empty
-- strings are dropped, a leading one is kept, and a string without the
-- separator is returned whole ("" gives {""}). An empty separator splits
-- into single chars.
function J.split(s, sep)
  need_string(s, "split")
  need_string(sep, "split")
  if s == "" then
    return { "" }
  end
  local parts = {}
  if sep == "" then
    for i = 1, #s do
      parts[i] = ssub(s, i, i)
    end
    return parts
  end
  local init = 1
  while true do
    local i, j = sfind(s, sep, init, true)
    if not i then
      break
    end
    parts[#parts + 1] = ssub(s, init, i - 1)
    init = j + 1
  end
  if #parts == 0 then
    return { s }
  end
  parts[#parts + 1] = ssub(s, init)
  while #parts > 0 and parts[#parts] == "" do
    parts[#parts] = nil
  end
  return parts
end

-- J.split_ws(s[, single]) is String.split("\\s+"): runs of ASCII
-- whitespace (space, \t, \n, \v, \f, \r) separate, with split's rules for
-- empty strings (a leading one is kept, trailing ones dropped). With
-- single true it is split("\\s"): each whitespace char separates.
function J.split_ws(s, single)
  need_string(s, "split")
  local ws = single and "[ \t\n\v\f\r]" or "[ \t\n\v\f\r]+"
  local parts, init = {}, 1
  while true do
    local i, j = sfind(s, ws, init)
    if not i then
      break
    end
    parts[#parts + 1] = ssub(s, init, i - 1)
    init = j + 1
  end
  if #parts == 0 then
    return { s }
  end
  parts[#parts + 1] = ssub(s, init)
  while #parts > 0 and parts[#parts] == "" do
    parts[#parts] = nil
  end
  return parts
end

-------------------------------------------------------------------------------
-- Arrays and lists (1-based Lua sequences)
-------------------------------------------------------------------------------

local function make_array(def, dims, d)
  local n = dims[d]
  if not is_integer(n) or n < 0 then
    throw("NegativeArraySizeException", tostring(n))
  end
  local t = {}
  for i = 1, n do
    if d < #dims then
      t[i] = make_array(def, dims, d + 1)
    else
      t[i] = def
    end
  end
  return t
end

-- J.array(default, n[, m...]) is new T[n][m]...: nested arrays filled with
-- default (0 for numbers, false for booleans).
function J.array(default, ...)
  local dims = { ... }
  if #dims == 0 then
    error("J.array needs at least one size", CALLER)
  end
  return make_array(default, dims, 1)
end

-- J.copy(t) is a shallow copy: array.clone(), list.toArray(), and the copy
-- constructors (new ArrayList<>(list), new HashMap<>(map), ...).
function J.copy(t)
  need_table(t, "copy")
  local c = {}
  for k, v in pairs(t) do
    c[k] = v
  end
  return c
end

-- J.copy_of(arr, n[, default]) is Arrays.copyOf: the first n elements, padded
-- with default when n is longer. Without default, the padding is 0 for
-- number arrays, false for boolean arrays and nil (Java's null) otherwise.
function J.copy_of(arr, n, default)
  need_table(arr, "copyOf")
  if not is_integer(n) or n < 0 then
    throw("NegativeArraySizeException", tostring(n))
  end
  if default == nil then
    local first = arr[1]
    if type(first) == "number" then
      default = 0
    elseif type(first) == "boolean" then
      default = false
    end
  end
  local c = {}
  for i = 1, n do
    local v = arr[i]
    if v == nil then
      v = default
    end
    c[i] = v
  end
  return c
end

-- J.fill(arr, value) is Arrays.fill: every element becomes value.
function J.fill(arr, value)
  need_table(arr, "fill")
  for i = 1, #arr do
    arr[i] = value
  end
end

-- random_below returns a random integer in 0 .. n-1, through the engine's
-- rnd when it exists.
local function random_below(n)
  local r = rnd
  if r ~= nil then
    return r(n)
  end
  return math.random(n) - 1
end

-- J.shuffle(list) is Collections.shuffle: a uniform random permutation, in
-- place.
function J.shuffle(list)
  need_table(list, "shuffle")
  for i = #list, 2, -1 do
    local j = random_below(i) + 1
    list[i], list[j] = list[j], list[i]
  end
end

-- J.contains(seq, value) is List.contains and Mobius ArrayUtil.contains:
-- whether an element equals value.
function J.contains(seq, value)
  need_table(seq, "contains")
  for i = 1, #seq do
    if seq[i] == value then
      return true
    end
  end
  return false
end

-- J.index_of_value(list, value) is List.indexOf: the 0-based index of the
-- first equal element, or -1.
function J.index_of_value(list, value)
  need_table(list, "indexOf")
  for i = 1, #list do
    if list[i] == value then
      return i - 1
    end
  end
  return -1
end

-- J.add(list, value) is List.add used as a value: appends and returns true.
function J.add(list, value)
  need_table(list, "add")
  list[#list + 1] = value
  return true
end

-- J.add_all(list, other) is List.addAll: appends the elements of other (a
-- list, or a set, whose elements are added in sorted order) and returns
-- whether the list changed.
function J.add_all(list, other)
  need_table(list, "addAll")
  need_table(other, "addAll")
  local n = #list
  if other[1] ~= nil or next(other) == nil then
    for i = 1, #other do
      list[#list + 1] = other[i]
    end
  else
    for _, v in ipairs(J.keys(other)) do
      list[#list + 1] = v
    end
  end
  return #list > n
end

-- J.remove_value(list, value) is List.remove(Object): removes the first
-- equal element and returns whether there was one.
function J.remove_value(list, value)
  need_table(list, "remove")
  for i = 1, #list do
    if list[i] == value then
      tremove(list, i)
      return true
    end
  end
  return false
end

-- element_string prints a collection element as Java's String.valueOf
-- would; numbers without a fraction print as integers.
local function element_string(v, seen)
  local tv = type(v)
  if v == nil then
    return "null"
  elseif tv == "number" then
    if is_integer(v) then
      return integral_string(v)
    end
    return J.str(v)
  elseif tv == "table" then
    if seen[v] then
      return "(this Collection)"
    end
    seen[v] = true
    local parts = {}
    for i = 1, #v do
      parts[i] = element_string(v[i], seen)
    end
    seen[v] = nil
    return "[" .. tconcat(parts, ", ") .. "]"
  end
  return tostring(v)
end

-- J.list_to_string(list) is List.toString: "[a, b, c]".
function J.list_to_string(list)
  need_table(list, "toString")
  return element_string(list, {})
end

-------------------------------------------------------------------------------
-- Maps and sets (Lua tables; a set maps each element to true)
-------------------------------------------------------------------------------

local type_rank = { number = 1, string = 2, boolean = 3 }

-- key_less orders keys: numbers, then strings, then booleans, then other
-- values by their tostring.
local function key_less(a, b)
  local ta, tb = type(a), type(b)
  if ta ~= tb then
    return (type_rank[ta] or 4) < (type_rank[tb] or 4) or
      ((type_rank[ta] or 4) == (type_rank[tb] or 4) and ta < tb)
  end
  if ta == "number" or ta == "string" then
    return a < b
  elseif ta == "boolean" then
    return b and not a
  end
  return tostring(a) < tostring(b)
end

-- J.keys(t) is map.keySet() (or a set's elements) as a list, in sorted key
-- order.
function J.keys(t)
  need_table(t, "keySet")
  local keys = {}
  for k in pairs(t) do
    keys[#keys + 1] = k
  end
  tsort(keys, key_less)
  return keys
end

-- J.values(t) is map.values() as a list, in sorted key order.
function J.values(t)
  need_table(t, "values")
  local keys = J.keys(t)
  for i, k in ipairs(keys) do
    keys[i] = t[k]
  end
  return keys
end

-- J.spairs(t) iterates a map (key, value) or a set (element, true) in
-- sorted key order, so loops over a Java HashMap give the same result every
-- time. Keys removed during the loop are skipped.
function J.spairs(t)
  need_table(t, "iteration")
  local keys = J.keys(t)
  local i = 0
  return function()
    while true do
      i = i + 1
      local k = keys[i]
      if k == nil then
        return nil
      end
      local v = t[k]
      if v ~= nil then
        return k, v
      end
    end
  end
end

-- J.size(t) is Map.size and Set.size: the number of entries.
function J.size(t)
  need_table(t, "size")
  local n = 0
  for _ in pairs(t) do
    n = n + 1
  end
  return n
end

-- J.clear(t) is clear() on a list, map or set: removes every entry.
function J.clear(t)
  need_table(t, "clear")
  for k in pairs(t) do
    t[k] = nil
  end
end

-- J.put(map, key, value) is Map.put used as a value: stores value and
-- returns the previous one (nil if none).
function J.put(map, key, value)
  need_table(map, "put")
  local old = map[key]
  map[key] = value
  return old
end

-- J.put_if_absent(map, key, value) is Map.putIfAbsent: stores value only if
-- key has none, and returns the value that was there (nil if none).
function J.put_if_absent(map, key, value)
  need_table(map, "putIfAbsent")
  local old = map[key]
  if old == nil then
    map[key] = value
  end
  return old
end

-- J.remove_key(map, key) is Map.remove used as a value: removes key and
-- returns its value (nil if none).
function J.remove_key(map, key)
  need_table(map, "remove")
  local old = map[key]
  map[key] = nil
  return old
end

-- J.get_or_default(map, key, default) is Map.getOrDefault.
function J.get_or_default(map, key, default)
  need_table(map, "getOrDefault")
  local v = map[key]
  if v == nil then
    return default
  end
  return v
end

-- J.contains_value(map, value) is Map.containsValue.
function J.contains_value(map, value)
  need_table(map, "containsValue")
  for _, v in pairs(map) do
    if v == value then
      return true
    end
  end
  return false
end

-- J.set_of(list) is new HashSet<>(collection): a set of the list's elements.
function J.set_of(list)
  need_table(list, "HashSet")
  local s = {}
  for i = 1, #list do
    s[list[i]] = true
  end
  return s
end

-- J.set_add(set, value) is Set.add used as a value: adds value and returns
-- whether it was new.
function J.set_add(set, value)
  need_table(set, "add")
  if set[value] ~= nil then
    return false
  end
  set[value] = true
  return true
end

-- J.set_remove(set, value) is Set.remove used as a value: removes value and
-- returns whether it was there.
function J.set_remove(set, value)
  need_table(set, "remove")
  if set[value] == nil then
    return false
  end
  set[value] = nil
  return true
end

-------------------------------------------------------------------------------
-- World objects kept in fields
-------------------------------------------------------------------------------

-- J.oid(obj) is what a quest field keeps for a world object: its object ID
-- (nil stays nil), since Player and Npc values expire when the handler
-- ends.
function J.oid(obj)
  if obj == nil or type(obj) == "number" then
    return obj
  end
  return obj:object_id()
end

-- J.obj(id) reads a field written by J.oid back as the world object, or nil
-- when the field is nil or the object is gone (world.find_object).
function J.obj(id)
  if id == nil or type(id) ~= "number" then
    return id
  end
  local w = world
  if w == nil or w.find_object == nil then
    error("J.obj needs world.find_object, which this server does not have yet", CALLER)
  end
  return w.find_object(id)
end

local player_classes = { Player = true, Playable = true, Creature = true, WorldObject = true, Object = true }
local npc_classes = { Npc = true, Creature = true, WorldObject = true, Object = true }
local not_npc_classes = { Player = true, Playable = true, Summon = true, Pet = true, Servitor = true }

-- J.instance_of(x, class) is Java's x instanceof Class for the classes
-- quests test. nil is never an instance. Players are Player, Playable,
-- Creature and WorldObject. NPCs are Npc, Creature and WorldObject; other
-- NPC classes (Attackable, Monster, VillageMasterPriest...) are answered by
-- npc:is_type(class).
function J.instance_of(x, class)
  local tx = type(x)
  if x == nil then
    return false
  elseif tx == "string" then
    return class == "String" or class == "CharSequence" or class == "Object"
  elseif tx == "number" then
    return class == "Number" or class == "Integer" or class == "Long" or class == "Double" or class == "Object"
  elseif tx == "boolean" then
    return class == "Boolean" or class == "Object"
  elseif tx == "userdata" then
    if x.class_id ~= nil then -- a Player
      return player_classes[class] == true
    end
    if x.id ~= nil and x.object_id ~= nil then -- an Npc
      if npc_classes[class] then
        return true
      end
      if not_npc_classes[class] then
        return false
      end
      if x.is_type ~= nil then
        return x:is_type(class) == true
      end
      error("J.instance_of: cannot tell whether an NPC is a " .. tostring(class) .. " (needs npc:is_type)", CALLER)
    end
  end
  error("J.instance_of: cannot test " .. tostring(x) .. " against " .. tostring(class), CALLER)
end

return J
