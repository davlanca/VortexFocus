#pragma once

#include <JuceHeader.h>
#include "PluginProcessor.h"

class VortexFocusAudioProcessorEditor  : public juce::AudioProcessorEditor
{
public:
    VortexFocusAudioProcessorEditor (VortexFocusAudioProcessor&);
    ~VortexFocusAudioProcessorEditor() override;

    void paint (juce::Graphics&) override;
    void resized() override;

private:
    VortexFocusAudioProcessor& audioProcessor;

    JUCE_DECLARE_NON_COPYABLE_WITH_LEAK_DETECTOR (VortexFocusAudioProcessorEditor)
};