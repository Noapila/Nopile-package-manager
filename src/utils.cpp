#include "nopile.hpp"
#include <iostream>
#include <iomanip>

// ── ANSI colors ────────────────────────────────────────────────────────────────

void print_error(const std::string& msg) {
    std::cerr << "\033[1;31m✗ Error:\033[0m " << msg << '\n';
}

void print_ok(const std::string& msg) {
    std::cout << "\033[1;32m✔\033[0m " << msg << '\n';
}

void print_info(const std::string& msg) {
    std::cout << "\033[1;34m→\033[0m " << msg << '\n';
}

// ── Paths ──────────────────────────────────────────────────────────────────────

fs::path pkg_file_path(const std::string& name) {
    return fs::path(NOPILE_PKG_DIR) / (name + ".nopile");
}

bool is_installed(const std::string& name) {
    return fs::exists(pkg_file_path(name));
}
