# FEAT-082: Agent Template Inheritance

## Overview
Implement a template inheritance system for agents to reduce duplication, ensure consistency, and make agent creation/maintenance more efficient. Create base templates that specialized agents can extend.

## Problem Statement
Current agent definitions have significant duplication:
- Same verification workflow repeated in every agent
- Identical completion requirements copy-pasted
- Common tool preferences duplicated
- Shared behaviors implemented inconsistently
- Maintenance requires updating multiple files
- New agents require copying boilerplate

## Requirements

### Functional Requirements
1. **Base Agent Templates**
   - **base-implementation-agent**: For agents that modify code
   - **base-review-agent**: For agents that review/analyze
   - **base-documentation-agent**: For agents that update docs
   - **base-orchestration-agent**: For agents that coordinate others

2. **Inheritance Mechanism**
   - Agents can extend base templates
   - Override specific sections as needed
   - Merge tool lists from parent and child
   - Inherit model and color defaults
   - Compose multiple behaviors

3. **Shared Behaviors**
   - Verification workflow requirements
   - Completion protocol (from FEAT-077)
   - Git commit standards
   - Error handling patterns
   - Tool usage preferences
   - Progress reporting

4. **Template Components**
   ```yaml
   base-template:
     core-behaviors:
       - completion-protocol
       - verification-workflow
       - commit-standards
     required-tools:
       - [list of tools]
     model: default-model
     color: default-color
   
   specialized-agent:
     extends: base-template
     additional-tools:
       - [specialized tools]
     overrides:
       model: different-model
     custom-behaviors:
       - [specialized behaviors]
   ```

### Non-Functional Requirements
1. **Maintainability**: Single source of truth for common behaviors
2. **Consistency**: Uniform behavior across similar agents
3. **Flexibility**: Easy to override and customize
4. **Clarity**: Clear inheritance hierarchy

## Design Approach

### Template Structure
```markdown
.claude/
├── agents/
│   ├── templates/
│   │   ├── base-implementation-agent.md
│   │   ├── base-review-agent.md
│   │   ├── base-documentation-agent.md
│   │   └── base-orchestration-agent.md
│   ├── spec-implementation-engineer.md
│   ├── product-doc-sync.md
│   └── ...
```

### Base Implementation Agent Template
```markdown
---
name: base-implementation-agent
type: template
tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS, TodoWrite
model: sonnet
color: green
---

## Core Behaviors

### Verification Workflow (MANDATORY)
[Standard verification requirements]

### Completion Protocol (MANDATORY)
[From FEAT-077]

### Git Standards (MANDATORY)
[Commit requirements]

### Tool Preferences
- Use Serena MCP for code analysis
- Fallback to basic tools only when necessary

## Required Capabilities
- Code modification
- Test execution
- Quality verification
- Progress tracking
```

### Specialized Agent Using Template
```markdown
---
name: spec-implementation-engineer
extends: base-implementation-agent
additional-tools: mcp__serena__*
model: opus  # Override
description: [specific description]
---

## Specialized Behaviors
[Only the unique aspects of this agent]

## Additional Responsibilities
[Specific to spec implementation]
```

### Template Processing
```python
def load_agent(agent_file):
    agent_def = parse_yaml(agent_file)
    
    if 'extends' in agent_def:
        base = load_template(agent_def['extends'])
        agent_def = merge_definitions(base, agent_def)
    
    return compile_agent(agent_def)

def merge_definitions(base, child):
    # Merge tools (union)
    # Override model/color if specified
    # Append behaviors
    # Merge descriptions
    return merged
```

## Tasks
- [ ] Design template inheritance system
- [ ] Create base-implementation-agent template
- [ ] Create base-review-agent template
- [ ] Create base-documentation-agent template
- [ ] Create base-orchestration-agent template
- [ ] Implement template processing logic
- [ ] Refactor existing agents to use templates
- [ ] Add validation for template references
- [ ] Create template composition system
- [ ] Document template usage guidelines
- [ ] Add tests for inheritance behavior
- [ ] Create agent creation wizard using templates

## Testing Requirements
1. **Unit Tests**
   - Test template loading and parsing
   - Verify inheritance merging logic
   - Test override behaviors

2. **Integration Tests**
   - Test agents using templates work correctly
   - Verify inherited behaviors function
   - Test multi-level inheritance

3. **Validation Tests**
   - Ensure all agents have required sections
   - Verify no circular dependencies
   - Check template references are valid

## Acceptance Criteria
- [ ] All base templates created and documented
- [ ] Existing agents refactored to use templates
- [ ] 70%+ reduction in agent definition duplication
- [ ] Template inheritance works correctly
- [ ] Override mechanism functions properly
- [ ] Documentation includes template hierarchy diagram
- [ ] New agent creation time reduced by 50%
- [ ] All agents maintain current functionality

## Dependencies
- FEAT-077: Agent Completion Protocol (will be included in templates)

## Priority
MEDIUM - Improves maintainability and consistency

## Estimated Effort
- Implementation: 6-8 hours
- Refactoring: 4-5 hours
- Testing: 2-3 hours
- Documentation: 2 hours
- Total: 14-18 hours

## Notes
- Consider YAML anchors for additional reuse
- May want to support multiple inheritance
- Could add template validation CLI command
- Future: Dynamic template generation based on requirements
- Consider versioning templates for backward compatibility