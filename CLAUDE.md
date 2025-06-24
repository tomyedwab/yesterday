Here are instructions for developing the Yesterday codebase.

## Design docs

Detailed designs for the project and its codebase are found in the `design`
directory, starting in `design/main.md`. These are maintained by the project
manager and will evolve over time.

## Spec files

Technical specifications for the codebase are derived strictly from the design
documents and live in the `spec` directory. Any time designs change, the
corresponding specifications should be updated accordingly.

Any TODOs or ambiguity in the design documents should be left as TODOs in the
spec files as well. DO NOT MAKE DECISIONS ON YOUR OWN - CLARIFICATIONS SHOULD GO
IN THE DESIGN DOCUMENTS.

Spec files need to reference the design documents by their full path, e.g.,
`design/map.md`. Specs include an introduction, reference matter and a list of
detailed implementation tasks along with their implementation status, for
example:

## Task `server-main`: Server main entrypoint
Reference: design/erver.md
Implementation status: Not implemented
Details:
(pseudocode goes here, along with example structures, references to files to create/edit, etc.)

Task IDs (such as `server-main` here) can be referenced in other tasks or in
the code implementation.

## Implementation

Each game system should go in its own directory, with a `README.md` file
referencing the relevant design document and specs.

Each file should have documentation at the top of the file, describing its
purpose, inputs, outputs, and any other relevant information with specific
references to the spec files.

Each file should have a corresponding tests file implementing detailed unit
tests for the system.

Each function should link to the relevant task[s] in the spec files that it
relates to in its docstring.

AFTER EACH TASK IS COMPLETED, DO THE FOLLOWING ACTIONS:

1) Run the unit tests and fix any failures. DO NOT COMMIT CHANGES UNLESS TESTS
ARE PASSING.

2) Update the spec file to reflect the updated task implementation status and
add file references or other details that will be helpful for future
development.

3) Commit all changed files to the repository.