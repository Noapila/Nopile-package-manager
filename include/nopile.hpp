#pragma once

#include <string>
#include <vector>
#include <map>
#include <filesystem>
#include <optional>

namespace fs = std::filesystem;

// ── Constantes ────────────────────────────────────────────────────────────────
inline constexpr const char* NOPILE_PKG_DIR    = "/var/lib/nopile/packages";
inline constexpr const char* NOPILE_CACHE_DIR  = "/var/cache/nopile";
inline constexpr const char* NOPILE_REPO_URL   =
    "https://raw.githubusercontent.com/Noapila/Nopile-package-manager-repo/main";
inline constexpr const char* NOPILE_VERSION    = "0.1.0";

// ── Structures ─────────────────────────────────────────────────────────────────

struct PackageMeta {
    std::string name;
    std::string version;
    std::string description;
    std::string maintainer;
    std::string arch;                     // x86_64, aarch64, …
    std::vector<std::string> depends;
    std::vector<std::string> files;       // liste des fichiers installés

    // Champs spécifiques à --compile
    std::string source_url;               // tarball sources
    std::string build_system;             // cmake | meson | autotools | make
    std::string configure_flags;
};

struct InstallOptions {
    bool compile   = false;   // --compile  : build depuis les sources
    bool force     = false;   // --force    : réinstaller même si présent
    bool no_deps   = false;   // --no-deps  : ignorer les dépendances
    bool verbose   = false;   // --verbose
};

// ── Fonctions publiques ────────────────────────────────────────────────────────

// Lecture / écriture de fichiers .nopile
PackageMeta parse_nopile_file(const fs::path& path);
void        write_nopile_file(const fs::path& path, const PackageMeta& meta);

// Gestion du dépôt distant
std::optional<PackageMeta> fetch_package_meta(const std::string& name);
bool download_file(const std::string& url, const fs::path& dest);

// Opérations
int cmd_install(const std::string& name, const InstallOptions& opts);
int cmd_remove (const std::string& name, bool purge = false);
int cmd_update ();
int cmd_list   ();
int cmd_info   (const std::string& name);
int cmd_search (const std::string& query);

// Helpers
bool        is_installed(const std::string& name);
fs::path    pkg_file_path(const std::string& name);
void        print_error  (const std::string& msg);
void        print_ok     (const std::string& msg);
void        print_info   (const std::string& msg);
