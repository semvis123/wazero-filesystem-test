#include "bridge.h"
#include <filesystem>

bool FileExists(char *filepath) { return std::filesystem::exists(filepath); }
