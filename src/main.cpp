#include "nopile.hpp"
#include <iostream>
#include <unistd.h>
#include <string>
#include <vector>
#include <algorithm>

static void print_usage() {
    std::cout <<
"Nopile " << NOPILE_VERSION << " - package manager for LFS\n\n"
"Usage:\n"
"  nopile <command> [options] [package]\n\n"
"Commands:\n"
"  install <package>   Install a package (prebuilt binary by default)\n"
"  remove  <package>   Remove an installed package\n"
"  update              Upgrade all installed packages\n"
"  list                List installed packages\n"
"  info    <package>   Show package information\n"
"  search  <term>      Search the repository\n"
"  version             Print nopile version\n\n"
"Install options:\n"
"  --compile           Build the package from source\n"
"  --force             Reinstall even if already present\n"
"  --no-deps           Skip dependency resolution\n"
"  --verbose           Show detailed build output\n\n"
"Examples:\n"
"  nopile install bash\n"
"  nopile install vim --compile --verbose\n"
"  nopile remove bash\n"
"  nopile list\n"
"  nopile search curl\n"
"  nopile info gcc\n\n"
"Installed packages : " << NOPILE_PKG_DIR << "/<package>.nopile\n"
"Cache              : " << NOPILE_CACHE_DIR << "\n";
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        print_usage();
        return 0;
    }

    std::vector<std::string> args(argv + 1, argv + argc);
    const std::string cmd = args[0];

    InstallOptions opts;
    std::string pkg_name;

    for (size_t i = 1; i < args.size(); ++i) {
        if      (args[i] == "--compile")  opts.compile  = true;
        else if (args[i] == "--force")    opts.force    = true;
        else if (args[i] == "--no-deps")  opts.no_deps  = true;
        else if (args[i] == "--verbose")  opts.verbose  = true;
        else if (args[i][0] != '-')       pkg_name      = args[i];
    }

    auto need_root = [&]() {
        if (geteuid() != 0) {
            print_error("This command must be run as root.");
            std::exit(1);
        }
    };

    if (cmd == "install") {
        if (pkg_name.empty()) { print_error("No package specified."); return 1; }
        need_root();
        return cmd_install(pkg_name, opts);
    }
    else if (cmd == "remove" || cmd == "rm") {
        if (pkg_name.empty()) { print_error("No package specified."); return 1; }
        need_root();
        bool purge = std::find(args.begin(), args.end(), "--purge") != args.end();
        return cmd_remove(pkg_name, purge);
    }
    else if (cmd == "update") {
        need_root();
        return cmd_update();
    }
    else if (cmd == "list" || cmd == "ls") {
        return cmd_list();
    }
    else if (cmd == "info") {
        if (pkg_name.empty()) { print_error("No package specified."); return 1; }
        return cmd_info(pkg_name);
    }
    else if (cmd == "search") {
        if (pkg_name.empty()) { print_error("No search term provided."); return 1; }
        return cmd_search(pkg_name);
    }
    else if (cmd == "version" || cmd == "--version" || cmd == "-v") {
        std::cout << "nopile " << NOPILE_VERSION << '\n';
        return 0;
    }
    else if (cmd == "--help" || cmd == "-h" || cmd == "help") {
        print_usage();
        return 0;
    }
    else {
        print_error("Unknown command: " + cmd);
        print_usage();
        return 1;
    }
}
