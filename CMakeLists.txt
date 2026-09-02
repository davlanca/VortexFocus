cmake_minimum_required(VERSION 3.22)

project(VortexFocus VERSION 0.0.1)

include(FetchContent)
FetchContent_Declare(
    JUCE
    GIT_REPOSITORY https://github.com/juce-framework/JUCE.git
    GIT_TAG        9.0.1
)
FetchContent_MakeAvailable(JUCE)

juce_add_plugin(VortexFocus
    COMPANY_NAME "Vortex"
    PLUGIN_NAME "VortexFocus"
    FORMATS VST3
    PRODUCT_NAME "VortexFocus"
)

juce_add_binary_data(VortexFocusData
    SOURCES
        Source/ui/index.html
)

target_sources(VortexFocus
    PRIVATE
        Source/PluginProcessor.cpp
        Source/PluginProcessor.h
        Source/PluginEditor.cpp
        Source/PluginEditor.h
)

target_compile_definitions(VortexFocus
    PUBLIC
        JUCE_WEB_BROWSER=1
        JUCE_USE_CURL=0
        JUCE_VST3_CAN_REPLACE_VST2=0
)

target_link_libraries(VortexFocus
    PRIVATE
        VortexFocusData
        juce::juce_audio_utils
        juce::juce_gui_extra
    PUBLIC
        juce::juce_recommended_config_flags
        juce::juce_recommended_warning_flags
)
