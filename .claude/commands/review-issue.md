Use agent spec-review-engineer to review issue $ARGUMENTS. Ask any questions about the issue.  Make any suggestions for improvements. The goal is to ensure the issue is complete in terms of product readiness.  It is ok if there are open implementation questions, those will be completed later.

After completing the review and any improvements to the issue document, auto-commit the changes using the standard workflow:

1. Check git status before and after review work
2. Stage only files that were modified during the review process  
3. Commit with message format: "Review and update issue $ARGUMENTS

Completed specification review with improvements

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

Only commit if changes were actually made to issue files during the review process.
