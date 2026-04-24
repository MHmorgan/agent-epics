#!/usr/bin/env nu

# ae CLI blackbox test suite

use std/assert

let test_dir = ($env.FILE_PWD)
let repo_root = ($test_dir | path dirname)

# Build binary
let tmpdir = (mktemp -d)
let ae = $"($tmpdir)/ae"

print "Building ae binary..."
let build = (do { cd $repo_root; go build -o $ae . } | complete)
if $build.exit_code != 0 {
    print $"BUILD FAILED: ($build.stderr)"
    rm -rf $tmpdir
    exit 1
}
print "Build OK"
print ""

$env.AE_APPDIR = $tmpdir

mut pass = 0
let total = 46

# ============================================================================
# Phase 1: Creation
# ============================================================================
print "--- Phase 1: Creation ---"

# task:new-epic myepic
try {
    let r = (^$ae task:new-epic myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:new-epic myepic"
} catch {|e|
    print $"FAIL: task:new-epic myepic — ($e.msg)"
}

# task:new-epic myepic2
try {
    let r = (^$ae task:new-epic myepic2 | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:new-epic myepic2"
} catch {|e|
    print $"FAIL: task:new-epic myepic2 — ($e.msg)"
}

# epics
try {
    let r = (^$ae epics | complete)
    let expected = (open --raw $"($test_dir)/epics-list.md")
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $expected "epics output"
    $pass += 1; print "PASS: epics list"
} catch {|e|
    print $"FAIL: epics list — ($e.msg)"
}

# ============================================================================
# Phase 2: Write operations
# ============================================================================
print ""
print "--- Phase 2: Write operations ---"

# task:set-body myepic (pipe show-body.md)
let body_fixture = (open --raw $"($test_dir)/show-body.md")
try {
    let r = ($body_fixture | ^$ae task:set-body myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:set-body myepic"
} catch {|e|
    print $"FAIL: task:set-body myepic — ($e.msg)"
}

# show myepic
try {
    let r = (^$ae show myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $body_fixture "body matches fixture"
    $pass += 1; print "PASS: show myepic"
} catch {|e|
    print $"FAIL: show myepic — ($e.msg)"
}

# task:set-context myepic
try {
    let r = ("This is the root context." | ^$ae task:set-context myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:set-context myepic"
} catch {|e|
    print $"FAIL: task:set-context myepic — ($e.msg)"
}

# context myepic
try {
    let r = (^$ae context myepic | complete)
    let expected_ctx = (open --raw $"($test_dir)/context-root.md")
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $expected_ctx "context matches fixture"
    $pass += 1; print "PASS: context myepic"
} catch {|e|
    print $"FAIL: context myepic — ($e.msg)"
}

# task:record myepic
try {
    let r = ("test record entry" | ^$ae task:record myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:record myepic"
} catch {|e|
    print $"FAIL: task:record myepic — ($e.msg)"
}

# task:records myepic
try {
    let r = (^$ae task:records myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    let agent_recs = ($r.stdout | from json | get data | where source == "agent")
    assert equal ($agent_recs | length) 1 "one agent record"
    assert equal ($agent_recs | first | get text) "test record entry" "record text"
    $pass += 1; print "PASS: task:records myepic (agent record)"
} catch {|e|
    print $"FAIL: task:records myepic — ($e.msg)"
}

# task myepic
try {
    let r = (^$ae task myepic | complete)
    let task_data = ($r.stdout | from json)
    assert equal $r.exit_code 0 "exit code"
    assert equal $task_data.ok true "ok"
    assert equal $task_data.data.id "myepic" "id"
    assert equal $task_data.data.is_leaf true "is_leaf"
    $pass += 1; print "PASS: task myepic (is_leaf=true)"
} catch {|e|
    print $"FAIL: task myepic — ($e.msg)"
}

# ============================================================================
# Phase 3: Structure
# ============================================================================
print ""
print "--- Phase 3: Structure ---"

# task:split myepic
try {
    let r = (^$ae task:split myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:split myepic"
} catch {|e|
    print $"FAIL: task:split myepic — ($e.msg)"
}

# task:list myepic
try {
    let r = (^$ae task:list myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    let ids = ($r.stdout | from json | get data | get id)
    assert equal $ids ["myepic", "myepic:1", "myepic:2"] "task ids"
    $pass += 1; print "PASS: task:list myepic"
} catch {|e|
    print $"FAIL: task:list myepic — ($e.msg)"
}

# task:list myepic parent=myepic
try {
    let r = (^$ae task:list myepic parent=myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:list myepic parent=myepic"
} catch {|e|
    print $"FAIL: task:list myepic parent=myepic — ($e.msg)"
}

# task myepic (now a branch)
try {
    let r = (^$ae task myepic | complete)
    let task_data = ($r.stdout | from json)
    assert equal $r.exit_code 0 "exit code"
    assert equal $task_data.ok true "ok"
    assert equal $task_data.data.is_leaf false "is_leaf"
    $pass += 1; print "PASS: task myepic (is_leaf=false after split)"
} catch {|e|
    print $"FAIL: task myepic is_leaf=false — ($e.msg)"
}

# Set body on myepic:1
let child_body = (open --raw $"($test_dir)/show-child-body.md")
try {
    let r = ($child_body | ^$ae task:set-body "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:set-body myepic:1"
} catch {|e|
    print $"FAIL: task:set-body myepic:1 — ($e.msg)"
}

# show myepic:1
try {
    let r = (^$ae show "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $child_body "body matches fixture"
    $pass += 1; print "PASS: show myepic:1"
} catch {|e|
    print $"FAIL: show myepic:1 — ($e.msg)"
}

# task:split myepic:1
try {
    let r = (^$ae task:split "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:split myepic:1"
} catch {|e|
    print $"FAIL: task:split myepic:1 — ($e.msg)"
}

# task:add-child myepic
try {
    let r = (^$ae task:add-child myepic | complete)
    let add_data = ($r.stdout | from json)
    assert equal $r.exit_code 0 "exit code"
    assert equal $add_data.ok true "ok"
    assert equal $add_data.data.id "myepic:3" "new child id"
    $pass += 1; print "PASS: task:add-child myepic (id=myepic:3)"
} catch {|e|
    print $"FAIL: task:add-child myepic — ($e.msg)"
}

# task:after myepic:2 myepic:1
try {
    let r = (^$ae task:after "myepic:2" "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:after myepic:2 myepic:1"
} catch {|e|
    print $"FAIL: task:after myepic:2 myepic:1 — ($e.msg)"
}

# task:unafter myepic:2 myepic:1
try {
    let r = (^$ae task:unafter "myepic:2" "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:unafter myepic:2 myepic:1"
} catch {|e|
    print $"FAIL: task:unafter myepic:2 myepic:1 — ($e.msg)"
}

# task:next myepic
try {
    let r = (^$ae task:next myepic | complete)
    let next_data = ($r.stdout | from json)
    assert equal $r.exit_code 0 "exit code"
    assert equal $next_data.ok true "ok"
    assert ($next_data.data != null) "data non-null"
    $pass += 1; print "PASS: task:next myepic (non-null)"
} catch {|e|
    print $"FAIL: task:next myepic — ($e.msg)"
}

# ============================================================================
# Phase 4: Context composition
# ============================================================================
print ""
print "--- Phase 4: Context composition ---"

# task:set-context myepic:1
try {
    let r = ("This is child 1 context." | ^$ae task:set-context "myepic:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:set-context myepic:1"
} catch {|e|
    print $"FAIL: task:set-context myepic:1 — ($e.msg)"
}

# task:set-context myepic:1:1
try {
    let r = ("This is grandchild context." | ^$ae task:set-context "myepic:1:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:set-context myepic:1:1"
} catch {|e|
    print $"FAIL: task:set-context myepic:1:1 — ($e.msg)"
}

# context myepic:1:1
try {
    let r = (^$ae context "myepic:1:1" | complete)
    let expected_composed = (open --raw $"($test_dir)/context-composed.md")
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $expected_composed "composed context matches fixture"
    $pass += 1; print "PASS: context myepic:1:1 (composed)"
} catch {|e|
    print $"FAIL: context myepic:1:1 — ($e.msg)"
}

# ============================================================================
# Phase 5: Status transitions
# ============================================================================
print ""
print "--- Phase 5: Status transitions ---"

# task:start myepic:1:1
try {
    let r = (^$ae task:start "myepic:1:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:start myepic:1:1"
} catch {|e|
    print $"FAIL: task:start myepic:1:1 — ($e.msg)"
}

# task:block myepic:1:1 "waiting for dep"
try {
    let r = (^$ae task:block "myepic:1:1" "waiting for dep" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:block myepic:1:1"
} catch {|e|
    print $"FAIL: task:block myepic:1:1 — ($e.msg)"
}

# task:unblock myepic:1:1
try {
    let r = (^$ae task:unblock "myepic:1:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:unblock myepic:1:1"
} catch {|e|
    print $"FAIL: task:unblock myepic:1:1 — ($e.msg)"
}

# task:done myepic:1:1
try {
    let r = (^$ae task:done "myepic:1:1" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:done myepic:1:1"
} catch {|e|
    print $"FAIL: task:done myepic:1:1 — ($e.msg)"
}

# task:start myepic:1:2
try {
    let r = (^$ae task:start "myepic:1:2" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:start myepic:1:2"
} catch {|e|
    print $"FAIL: task:start myepic:1:2 — ($e.msg)"
}

# task:abandon myepic:1:2 "no longer needed"
try {
    let r = (^$ae task:abandon "myepic:1:2" "no longer needed" | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:abandon myepic:1:2"
} catch {|e|
    print $"FAIL: task:abandon myepic:1:2 — ($e.msg)"
}

# ============================================================================
# Phase 6: Attributes
# ============================================================================
print ""
print "--- Phase 6: Attributes ---"

# attr:set myepic summary
try {
    let r = ("This is the summary value" | ^$ae attr:set myepic summary | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: attr:set myepic summary"
} catch {|e|
    print $"FAIL: attr:set myepic summary — ($e.msg)"
}

# attr:get myepic summary
try {
    let r = (^$ae attr:get myepic summary | complete)
    let attr_data = ($r.stdout | from json)
    assert equal $r.exit_code 0 "exit code"
    assert equal $attr_data.ok true "ok"
    assert equal $attr_data.data.value "This is the summary value" "value"
    $pass += 1; print "PASS: attr:get myepic summary"
} catch {|e|
    print $"FAIL: attr:get myepic summary — ($e.msg)"
}

# ============================================================================
# Phase 7: Error cases
# ============================================================================
print ""
print "--- Phase 7: Error cases ---"

# Duplicate epic
try {
    let r = (^$ae task:new-epic myepic | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — duplicate epic"
} catch {|e|
    print $"FAIL: error — duplicate epic — ($e.msg)"
}

# Set body on branch
try {
    let r = ("some body" | ^$ae task:set-body myepic | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — set-body on branch"
} catch {|e|
    print $"FAIL: error — set-body on branch — ($e.msg)"
}

# Split leaf with no body (myepic:3 has no body)
try {
    let r = (^$ae task:split "myepic:3" | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — split leaf without body"
} catch {|e|
    print $"FAIL: error — split leaf without body — ($e.msg)"
}

# Add child to leaf (myepic:3 is a leaf)
try {
    let r = (^$ae task:add-child "myepic:3" | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — add-child to leaf"
} catch {|e|
    print $"FAIL: error — add-child to leaf — ($e.msg)"
}

# Done on pending task (myepic:2 is pending)
try {
    let r = (^$ae task:done "myepic:2" | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — done on pending"
} catch {|e|
    print $"FAIL: error — done on pending — ($e.msg)"
}

# Get nonexistent attribute
try {
    let r = (^$ae attr:get myepic nonexistent | complete)
    assert equal $r.exit_code 1 "exit code"
    assert equal ($r.stdout | from json | get ok?) false "ok"
    $pass += 1; print "PASS: error — attr:get nonexistent"
} catch {|e|
    print $"FAIL: error — attr:get nonexistent — ($e.msg)"
}

# show nonexistent-epic
try {
    let r = (^$ae show nonexistent-epic | complete)
    assert ($r.exit_code != 0) "non-zero exit"
    $pass += 1; print "PASS: error — show nonexistent-epic"
} catch {|e|
    print $"FAIL: error — show nonexistent-epic — ($e.msg)"
}

# context nonexistent-epic
try {
    let r = (^$ae context nonexistent-epic | complete)
    assert ($r.exit_code != 0) "non-zero exit"
    $pass += 1; print "PASS: error — context nonexistent-epic"
} catch {|e|
    print $"FAIL: error — context nonexistent-epic — ($e.msg)"
}

# ============================================================================
# Phase 8: Cleanup commands
# ============================================================================
print ""
print "--- Phase 8: Cleanup commands ---"

# task:start myepic2
try {
    let r = (^$ae task:start myepic2 | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:start myepic2"
} catch {|e|
    print $"FAIL: task:start myepic2 — ($e.msg)"
}

# task:done myepic2
try {
    let r = (^$ae task:done myepic2 | complete)
    assert equal $r.exit_code 0 "exit code"
    assert equal ($r.stdout | from json | get ok?) true "ok"
    $pass += 1; print "PASS: task:done myepic2"
} catch {|e|
    print $"FAIL: task:done myepic2 — ($e.msg)"
}

# purge
try {
    let r = (^$ae purge | complete)
    assert equal $r.exit_code 0 "exit code"
    $pass += 1; print "PASS: purge"
} catch {|e|
    print $"FAIL: purge — ($e.msg)"
}

# epics after purge
try {
    let r = (^$ae epics | complete)
    let expected_after = (open --raw $"($test_dir)/epics-after-rm.md")
    assert equal $r.exit_code 0 "exit code"
    assert equal $r.stdout $expected_after "epics after purge"
    $pass += 1; print "PASS: epics after purge"
} catch {|e|
    print $"FAIL: epics after purge — ($e.msg)"
}

# rm myepic
try {
    let r = (^$ae rm myepic | complete)
    assert equal $r.exit_code 0 "exit code"
    $pass += 1; print "PASS: rm myepic"
} catch {|e|
    print $"FAIL: rm myepic — ($e.msg)"
}

# epics after rm (should be empty)
try {
    let r = (^$ae epics | complete)
    assert equal ($r.stdout | str trim) "" "empty output"
    assert equal $r.exit_code 0 "exit code"
    $pass += 1; print "PASS: epics empty after rm"
} catch {|e|
    print $"FAIL: epics empty after rm — ($e.msg)"
}

# ============================================================================
# Summary
# ============================================================================
print ""
let fail = $total - $pass
print $"Results: ($pass) passed, ($fail) failed"
rm -rf $tmpdir
if $fail > 0 { exit 1 }