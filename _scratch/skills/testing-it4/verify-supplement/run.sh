#!/bin/zsh
BASE=/Users/danielorbach/Code/modern-engineering/prototype/.claude/worktrees/master-tests-orig/_scratch/skills/testing-it4
OUT=$BASE/verify-supplement
RUNS=(
limiter-write-tests/with_skill_fable__r2
limiter-write-tests/with_skill_fable__r3
limiter-write-tests/with_skill_opus__r2
limiter-write-tests/with_skill_opus__r3
limiter-write-tests/with_skill_sonnet__r2
limiter-write-tests/with_skill_sonnet__r3
limiter-write-tests/with_skill_looped_fable__r1
limiter-write-tests/with_skill_looped_fable__r2
limiter-write-tests/with_skill_looped_fable__r3
limiter-write-tests/with_skill_looped_opus__r1
limiter-write-tests/with_skill_looped_opus__r2
limiter-write-tests/with_skill_looped_opus__r3
limiter-write-tests/with_skill_looped_sonnet__r2
limiter-write-tests/with_skill_looped_sonnet__r3
truncate-drift/with_skill_fable__r2
truncate-drift/with_skill_fable__r3
truncate-drift/with_skill_opus__r2
truncate-drift/with_skill_opus__r3
truncate-drift/with_skill_sonnet__r2
truncate-drift/with_skill_sonnet__r3
truncate-drift/with_skill_looped_fable__r1
truncate-drift/with_skill_looped_fable__r2
truncate-drift/with_skill_looped_fable__r3
truncate-drift/with_skill_looped_opus__r1
truncate-drift/with_skill_looped_opus__r2
truncate-drift/with_skill_looped_opus__r3
truncate-drift/with_skill_looped_sonnet__r2
truncate-drift/with_skill_looped_sonnet__r3
)
for r in $RUNS; do
  d=$BASE/iteration-4/$r/outputs
  slug=${r//\//-}
  log=$OUT/$slug.log
  {
    echo "== $r =="
    fixture=$BASE/fixtures/limiter/limiter.go; src=$d/limiter.go
    [[ $r == truncate-drift/* ]] && fixture=$BASE/fixtures/truncate/truncate.go && src=$d/truncate.go
    if diff -q $fixture $src >/dev/null 2>&1; then echo "SOURCE: unchanged"; else echo "SOURCE: MODIFIED"; diff -u $fixture $src | head -40; fi
    echo "--- files ---"; ls $d
    echo "--- go vet ---"; (cd $d && go vet ./... 2>&1 | head -20)
    echo "--- go test (single) ---"; (cd $d && go test -timeout 120s ./... 2>&1 | tail -30)
    echo "--- go test -count=30 ---"; (cd $d && go test -timeout 300s -count=30 ./... 2>&1 | tail -8)
    echo "--- go test -race -count=5 ---"; (cd $d && go test -timeout 300s -race -count=5 ./... 2>&1 | tail -8)
  } > $log 2>&1
  echo "done $r"
done
