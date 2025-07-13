# Contributing to Logma

We welcome contributions to Logma! To ensure a smooth and effective collaboration, please follow these guidelines.

## How to Contribute

1.  **Fork the Repository:** Start by forking the `logma` repository to your GitHub account.
2.  **Clone Your Fork:** Clone your forked repository to your local machine:
    ```bash
    git clone https://github.com/felixgeelhaar/logma.git
    cd logma
    ```
3.  **Create a New Branch:** Create a new branch for your feature or bug fix. Use a descriptive name:
    ```bash
    git checkout -b feature/your-feature-name
    # or
    git checkout -b bugfix/your-bug-fix
    ```
4.  **Implement Your Changes:** Make your changes, adhering to the project's coding style and conventions. Remember to follow the Test-Driven Development (TDD) methodology:
    *   Write a failing test that demonstrates the bug or the absence of the feature.
    *   Write the minimum code necessary to make the test pass.
    *   Refactor your code for clarity and efficiency.
5.  **Run Tests and Linters:** Before committing, ensure all tests pass and that your code adheres to Go's best practices and style guidelines:
    ```bash
    go test ./...
    go vet ./...
    golint ./...
    ```
6.  **Commit Your Changes:** Write clear, concise commit messages. Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification (e.g., `feat: Add new feature`, `fix: Fix a bug`).
    ```bash
    git commit -m "feat: Your descriptive commit message"
    ```
7.  **Push to Your Fork:** Push your new branch to your forked repository:
    ```bash
    git push origin feature/your-feature-name
    ```
8.  **Create a Pull Request (PR):** Go to the original `logma` repository on GitHub and create a new Pull Request from your forked branch to the `main` branch. Provide a clear description of your changes.

## Code Style

*   Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
*   Ensure your code is `go fmt` compliant.
*   Add GoDoc comments for all public types, functions, and methods.

## Reporting Bugs

If you find a bug, please open an issue on the [GitHub Issues page](https://github.com/felixgeelhaar/logma/issues). Provide a clear description of the bug, steps to reproduce it, and expected behavior.

## Feature Requests

If you have a feature idea, please open an issue on the [GitHub Issues page](https://github.com/felixgeelhaar/logma/issues) to discuss it. This helps ensure that the feature aligns with the project's goals and avoids duplicate work.
