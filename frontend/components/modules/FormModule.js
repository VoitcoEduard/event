import React, { useState } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, Alert, ScrollView, Platform } from 'react-native';
import { Ionicons } from '@expo/vector-icons';

// Use environment variable or default to localhost for development
const API_URL = typeof window !== 'undefined' && window.location
    ? (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
        ? 'http://localhost:8080'
        : `http://${window.location.hostname}:8080`)
    : 'http://localhost:8080';

export default function FormModule({ module, eventId, userId, onComplete }) {
    const timeline = module.data?.timeline || [];
    const [currentStepIndex, setCurrentStepIndex] = useState(0);
    const [answers, setAnswers] = useState({}); // { [stepId]: value }
    const [submitting, setSubmitting] = useState(false);
    const [completed, setCompleted] = useState(false);

    const currentStep = timeline[currentStepIndex];
    const isLastStep = currentStepIndex === timeline.length - 1;

    const handleNext = async () => {
        if (currentStep.type === 'field') {
            const answer = answers[currentStep.id];
            if (!answer || (typeof answer === 'string' && !answer.trim())) {
                Alert.alert('Validation', 'Please provide an answer.');
                return;
            }
        }

        if (isLastStep) {
            await handleSubmit();
        } else {
            setCurrentStepIndex(prev => prev + 1);
        }
    };

    const handleBack = () => {
        if (currentStepIndex > 0) {
            setCurrentStepIndex(prev => prev - 1);
        }
    };

    const handleSubmit = async () => {
        setSubmitting(true);
        try {
            const response = await fetch(`${API_URL}/events/${eventId}/modules/${module.id}/response`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    userId: userId,
                    data: answers
                }),
            });

            if (response.ok) {
                setCompleted(true);
                if (onComplete) onComplete();
            } else {
                Alert.alert('Error', 'Failed to submit response.');
            }
        } catch (error) {
            console.error('Error submitting response:', error);
            Alert.alert('Error', 'Failed to submit response.');
        } finally {
            setSubmitting(false);
        }
    };

    const updateAnswer = (text) => {
        setAnswers(prev => ({ ...prev, [currentStep.id]: text }));
    };

    if (completed) {
        return (
            <View style={styles.container}>
                <Ionicons name="checkmark-circle" size={80} color="#30D158" style={{ marginBottom: 20 }} />
                <Text style={styles.title}>All Done!</Text>
                <Text style={styles.text}>Your responses have been saved.</Text>
                <Text style={styles.subText}>Sit tight for the next activity.</Text>
            </View>
        );
    }

    if (!currentStep) {
        return (
            <View style={styles.container}>
                <Text style={styles.text}>Empty Form Configuration</Text>
            </View>
        );
    }

    const renderFieldInput = () => {
        const fieldType = currentStep.data.fieldType || 'text';
        const value = answers[currentStep.id] || '';

        switch (fieldType) {
            case 'number':
                return (
                    <TextInput
                        style={styles.input}
                        placeholder="Type a number..."
                        placeholderTextColor="#666"
                        value={value}
                        onChangeText={(text) => updateAnswer(text.replace(/[^0-9]/g, ''))}
                        keyboardType="numeric"
                    />
                );
            case 'date':
                return (
                    <TextInput
                        style={styles.input}
                        placeholder="YYYY-MM-DD"
                        placeholderTextColor="#666"
                        value={value}
                        onChangeText={updateAnswer}
                    />
                );
            case 'text':
            default:
                return (
                    <TextInput
                        style={[styles.input, { minHeight: 100, textAlignVertical: 'top' }]}
                        placeholder="Type your answer here..."
                        placeholderTextColor="#666"
                        value={value}
                        onChangeText={updateAnswer}
                        multiline
                    />
                );
        }
    };

    return (
        <View style={styles.container}>
            <View style={styles.progressBar}>
                <View style={[styles.progressFill, { width: `${((currentStepIndex + 1) / timeline.length) * 100}%` }]} />
            </View>

            <ScrollView contentContainerStyle={styles.scrollContent}>
                <Text style={styles.stepCounter}>Step {currentStepIndex + 1} of {timeline.length}</Text>

                {currentStep.type === 'message' ? (
                    <View style={styles.messageCard}>
                        <Text style={styles.messageText}>{currentStep.data.content}</Text>
                    </View>
                ) : (
                    <View style={styles.fieldCard}>
                        <Text style={styles.fieldLabel}>{currentStep.data.label}</Text>
                        {renderFieldInput()}
                    </View>
                )}
            </ScrollView>

            <View style={styles.footer}>
                {currentStepIndex > 0 && (
                    <TouchableOpacity style={styles.backButton} onPress={handleBack} disabled={submitting}>
                        <Text style={styles.backButtonText}>Back</Text>
                    </TouchableOpacity>
                )}
                <TouchableOpacity
                    style={[styles.nextButton, submitting && { opacity: 0.7 }]}
                    onPress={handleNext}
                    disabled={submitting}
                >
                    {submitting ? (
                        <ActivityIndicator color="#FFF" />
                    ) : (
                        <Text style={styles.nextButtonText}>{isLastStep ? 'Submit' : 'Next'}</Text>
                    )}
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        width: '100%',
        flex: 1,
        alignItems: 'center',
    },
    progressBar: {
        width: '100%',
        height: 6,
        backgroundColor: '#333',
        borderRadius: 3,
        marginBottom: 20,
        overflow: 'hidden',
    },
    progressFill: {
        height: '100%',
        backgroundColor: '#30D158',
    },
    scrollContent: {
        width: '100%',
        alignItems: 'center',
        paddingBottom: 100,
    },
    stepCounter: {
        color: '#666',
        marginBottom: 16,
        fontSize: 14,
        fontWeight: '600',
        textTransform: 'uppercase',
    },
    title: {
        fontSize: 28,
        fontWeight: 'bold',
        color: '#FFF',
        marginBottom: 10,
    },
    text: {
        fontSize: 18,
        color: '#CCC',
        marginBottom: 8,
        textAlign: 'center',
    },
    subText: {
        fontSize: 14,
        color: '#888',
        fontStyle: 'italic',
    },
    messageCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 30,
        width: '100%',
        borderWidth: 1,
        borderColor: '#333',
        alignItems: 'center',
    },
    messageText: {
        fontSize: 20,
        color: '#FFF',
        lineHeight: 30,
        textAlign: 'center',
    },
    fieldCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 24,
        width: '100%',
        borderWidth: 1,
        borderColor: '#333',
    },
    fieldLabel: {
        fontSize: 18,
        color: '#FFF',
        fontWeight: '700',
        marginBottom: 16,
    },
    input: {
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
        padding: 16,
        fontSize: 16,
        color: '#FFF',
        borderWidth: 1,
        borderColor: '#444',
    },
    footer: {
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        flexDirection: 'row',
        padding: 20,
        gap: 12,
        // backgroundColor: '#121212', // Optional: background for sticky footer
    },
    backButton: {
        flex: 1,
        paddingVertical: 16,
        borderRadius: 12,
        backgroundColor: '#333',
        alignItems: 'center',
    },
    backButtonText: {
        color: '#FFF',
        fontSize: 16,
        fontWeight: '600',
    },
    nextButton: {
        flex: 2,
        paddingVertical: 16,
        borderRadius: 12,
        backgroundColor: '#007AFF',
        alignItems: 'center',
    },
    nextButtonText: {
        color: '#FFF',
        fontSize: 16,
        fontWeight: '600',
    },
});
