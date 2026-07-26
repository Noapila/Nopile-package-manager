#include "nopile.hpp"

#include <openssl/evp.h>
#include <fstream>
#include <sstream>
#include <iomanip>

std::string md5_file(const fs::path& path) {
    std::ifstream f(path, std::ios::binary);
    if (!f) return "";

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_md5(), nullptr);

    char buf[8192];
    while (f.read(buf, sizeof(buf)) || f.gcount())
        EVP_DigestUpdate(ctx, buf, (size_t)f.gcount());

    unsigned char digest[16];
    unsigned int  len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);

    std::ostringstream ss;
    for (int i = 0; i < 16; ++i)
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)digest[i];
    return ss.str();
}
