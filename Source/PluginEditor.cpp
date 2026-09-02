#include "PluginProcessor.h"
#include "PluginEditor.h"

VortexFocusAudioProcessorEditor::VortexFocusAudioProcessorEditor (VortexFocusAudioProcessor& p)
    : AudioProcessorEditor (&p), audioProcessor (p)
{
    setSize(920, 820);
}

VortexFocusAudioProcessorEditor::~VortexFocusAudioProcessorEditor()
{
}

void VortexFocusAudioProcessorEditor::paint (juce::Graphics& g)
{
    g.fillAll (juce::Colours::black);
    g.setColour (juce::Colours::white);
    g.setFont (juce::FontOptions (24.0f));
    g.drawFittedText ("VortexFocus", getLocalBounds(), juce::Justification::centred, 1);
}

void VortexFocusAudioProcessorEditor::resized()
{
}