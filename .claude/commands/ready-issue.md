Use agent spec-review-engineer to read through issue $ARGUMENTS.  It needs to verify that there is enough detail to start the implementation.  

If the issue is ready for implementation:

1. **Check git status before starting** to capture initial state
2. **Move the issue file** using `git mv issues/backlog/[ISSUE].md issues/ready/[ISSUE].md`
3. **Verify the move completed correctly** with `git status` (should show renamed file)
4. **Commit the change** with proper message format:

```
marked [ISSUE-ID] ready

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Important**: 
- ONLY use `git mv` for the file movement - never copy and delete separately
- The final git status should show the file as renamed/moved, not as separate add/delete operations  
- If git mv fails or git status shows anything other than a clean rename, investigate before committing
- Do NOT leave modified files in backlog/ or uncommitted files in ready/

