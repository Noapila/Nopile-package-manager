#include "nopile.hpp"

#include <iostream>
#include <string>
#include <vector>
#include <unistd.h>

int main(int argc, char* argv[]) {
    if (argc < 2) {
        cmd_help();
        return 0;
    }

    std::vector<std::string> args(argv + 1, argv + argc);
    const std::string cmd = args[0];

    // nopile -i /path/to/file.tar.gz
    if (cmd == "-i") {
        if (args.size() < 2) {
            std::cerr << "[nopile] -i requires a path\n";
            return 1;
        }
        if (geteuid() != 0) {
            std::cerr << "[nopile] must be run as root\n";
            return 1;
        }
        bool compile = false;
        for (auto& a : args)
            if (a == "--compile") compile = true;
        return cmd_install("", compile, true, fs::path(args[1]));
    }

    if (cmd == "help" || cmd == "--help" || cmd == "-h") {
        return cmd_help();
    }

    if (cmd == "search") {
        if (args.size() < 2) {
            std::cerr << "[nopile] search requires a term\n";
            return 1;
        }
        return cmd_search(args[1]);
    }

    if (cmd == "update") {
        if (geteuid() != 0) {
            std::cerr << "[nopile] must be run as root\n";
            return 1;
        }
        return cmd_update();
    }

    if (cmd == "install") {
        if (args.size() < 2) {
            std::cerr << "[nopile] install requires a package name\n";
            return 1;
        }
        if (geteuid() != 0) {
            std::cerr << "[nopile] must be run as root\n";
            return 1;
        }
        bool compile = false;
        for (auto& a : args)
            if (a == "--compile") compile = true;
        return cmd_install(args[1], compile, false, {});
    }

    if (cmd == "remove") {
        if (args.size() < 2) {
            std::cerr << "[nopile] remove requires a package name\n";
            return 1;
        }
        if (geteuid() != 0) {
            std::cerr << "[nopile] must be run as root\n";
            return 1;
        }
        return cmd_remove(args[1]);
    }

    std::cerr << "[nopile] unknown command: " << cmd
              << " — try: nopile help\n";
    return 1;
}
