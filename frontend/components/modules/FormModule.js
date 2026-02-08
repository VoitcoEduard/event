import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, Alert, ScrollView, Platform, Modal } from 'react-native';
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
    const [templates, setTemplates] = useState([]);
    const [showTemplateModal, setShowTemplateModal] = useState(false);
    const [loadingTemplates, setLoadingTemplates] = useState(false);

    const currentStep = timeline[currentStepIndex];
    const isLastStep = currentStepIndex === timeline.length - 1;

    // Fetch templates on mount
    useEffect(() => {
        fetchTemplates();
    }, []);

    const fetchTemplates = async () => {
        setLoadingTemplates(true);
        try {
            const response = await fetch(`${API_URL}/templates`);
            if (response.ok) {
                const data = await response.json();
                setTemplates(data);
            } else {
                console.error('Failed to fetch templates');
            }
        } catch (error) {
            console.error('Error fetching templates:', error);
        } finally {
            setLoadingTemplates(false);
        }
    };

    const handleCloseTemplateModal = () => {
        setShowTemplateModal(false);
    };

    const handleShowTemplateModal = () => {
        setShowTemplateModal(true);
    };

    const handleLoadTemplate = (template) => {
        // Load template data into answers
        if (template.data && template.data.answers) {
            setAnswers(prev => ({ ...prev, ...template.data.answers }));
        }
        setShowTemplateModal(false);
        Alert.alert('Success', 'Template loaded successfully!');
    };

    const handleDeleteTemplate = async (id, name) => {
        Alert.alert(
            "Delete Template",
            `Are you sure you want to delete "${name}"?`,
            [
                { text: "Cancel", style: "cancel" },
                {
                    text: "Delete",
                    style: "destructive",
                    onPress: async () => {
                        try {
                            const response = await fetch(`${API_URL}/templates/${id}`, {
                                method: 'DELETE',
                            });
                            if (response.ok) {
                                fetchTemplates();
                            } else {
                                Alert.alert("Error", "Failed to delete template.");
                            }
                        } catch (error) {
                            console.error(error);
                            Alert.alert("Error", "Failed to delete template.");
                        }
                    }
                }
            ]
        );
    };

    const renderTemplateModal = () => {
        if (!showTemplateModal) return null;

        return (
            <Modal
                visible={showTemplateModal}
                transparent
                animationType="fade"
                onRequestClose={handleCloseTemplateModal}
            >
                <View style={styles.dialogOverlay}>
                    <View style={styles.dialog}>
                        <Text style={styles.dialogTitle}>Load from Template</Text>
                        <ScrollView style={{ maxHeight: 300 }}>
                            {loadingTemplates ? (
                                <ActivityIndicator size="small" color="#007AFF" />
                            ) : templates.map(t => (
                                <View key={t.id} style={styles.templateItemWrapper}>
                                    <TouchableOpacity style={styles.templateItem} onPress={() => handleLoadTemplate(t)}>
                                        <Text style={styles.templateItemName}>{t.name}</Text>
                                        <Text style={styles.templateItemDesc}>{t.description}</Text>
                                    </TouchableOpacity>
                                    <TouchableOpacity style={styles.deleteTemplateBtn} onPress={() => handleDeleteTemplate(t.id, t.name)}>
                                        <Ionicons name="trash-outline" size={20} color="#FF453A" />
                                    </TouchableOpacity>
                                </View>
                            ))}
                            {!loadingTemplates && templates.length === 0 && (
                                <Text style={{ color: '#666', textAlign: 'center', padding: 20 }}>No templates found.</Text>
                            )}
                        </ScrollView>
                        <TouchableOpacity style={styles.dialogCloseBtn} onPress={handleCloseTemplateModal}>
                            <Text style={styles.dialogCloseText}>Cancel</Text>
                        </TouchableOpacity>
                    </View>
                </View>
            </Modal>
        );
    };

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
            {renderTemplateModal()}

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

            {/* Template Load Button */}
            <TouchableOpacity style={styles.templateButton} onPress={handleShowTemplateModal}>
                <Ionicons name="folder-open-outline" size={18} color="#007AFF" />
                <Text style={styles.templateButtonText}>Load Template</Text>
            </TouchableOpacity>
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
    templateButton: {
        position: 'absolute',
        top: 10,
        right: 10,
        flexDirection: 'row',
        alignItems: 'center',
        paddingVertical: 8,
        paddingHorizontal: 12,
        backgroundColor: 'rgba(0, 122, 255, 0.1)',
        borderRadius: 8,
    },
    templateButtonText: {
        color: '#007AFF',
        fontSize: 14,
        fontWeight: '600',
        marginLeft: 6,
    },
    dialogOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0, 0, 0, 0.7)',
        justifyContent: 'center',
        alignItems: 'center',
    },
    dialog: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 24,
        width: '85%',
        maxWidth: 400,
    },
    dialogTitle: {
        fontSize: 20,
        fontWeight: 'bold',
        color: '#FFF',
        marginBottom: 16,
        textAlign: 'center',
    },
    templateItemWrapper: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 12,
    },
    templateItem: {
        flex: 1,
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
        padding: 16,
    },
    templateItemName: {
        fontSize: 16,
        fontWeight: '600',
        color: '#FFF',
    },
    templateItemDesc: {
        fontSize: 14,
        color: '#888',
        marginTop: 4,
    },
    deleteTemplateBtn: {
        padding: 8,
        marginLeft: 8,
    },
    dialogCloseBtn: {
        marginTop: 16,
        padding: 16,
        backgroundColor: '#333',
        borderRadius: 12,
        alignItems: 'center',
    },
    dialogCloseText: {
        color: '#FFF',
        fontSize: 16,
        fontWeight: '600',
    },
});
