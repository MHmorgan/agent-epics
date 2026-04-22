Agent Epics
===========

A tool for tracking agent tasks, based around a concept of "epics".

Workflow for a simple task:
- You start planning a task with the agent -> it creates a new epic


## Usage



## Commands

```
# Human interface
ae epics  # List all epics (top-level tasks)

# Agent interface

ae task new      "<task-id>" # Create a new task
ae task describe "<task-id>" "<description>"  # Set the description of an task
ae task show     "<task-id>" # Print the task

ae task section set     "<task-id>" "<section>" "<content>"  # Set the content of an task section
ae task section reorder "<task-id>" "<section>" <position>   # Set the section position
ae task section merge   "<task-id>" "<section1>" "<section2>"  # Merge section2 into section1

# Split an task. Each section becomes a sub-task with the section content as description
ae task split "<task-id>"

# Set a parent task which this task depends on
ae task depend "<task-id>" "<parent-task-id>"

# Record an action for an task (e.g. the agent deviated from the plan, something went wrong, etc.)
ae task record "<task-id>" "<details>"

# Set the summary for the task (a summary should be written after a task is complete)
ae task summary "<task-id>" "<summary>"
```

tasks are identified with a simple hierarchical structure: `my-epic`, `my-epic:first-task`, etc.
Top-level tasks are referred to as "epics".

tasks are either "split" or "unsplit".
An unsplit task is a leaf task without any sub-tasks, and can contain sections.
A split task is a branch with sub-tasks, but no sections.

Semantically each leaf task is intended to be the scope of one claude-code implementation session
(which may include a complex architecture of sub-agents or agent teams). It's one chunk of work.
This means leaf tasks should be sectioned sparingly - so that the splitting of tasks result
in a suitable layout of new sub-tasks.


## Internally
