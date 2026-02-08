import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ScrollView, Alert, Modal, Platform, Animated, Easing } from 'react-native';
import { Ionicons } from '@expo/vector-icons';

// Use environment variable or default to localhost for development
const API_URL = typeof window !== 'undefined' && window.location
    ? (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
        ? 'http://localhost:8080'
        : `http://${window.location.hostname}:8080`)
    : 'http://localhost:8080';

export default function FormModuleConfig({ visible, module, eventId, onClose, onUpdate }) {
    const [timeline, setTimeline] = useState([]);
    const [loading, setLoading] = useState(false);
    const [templates, setTemplates] = useState([]);
    const [showTemplateModal, setShowTemplateModal] = useState(false);
    const [templateName, setTemplateName] = useState('');
    const [showSaveTemplateModal, setShowSaveTemplateModal] = useState(false);

    // Animation for modal
    const [slideAnim] = useState(new Animated.Value(0));

    useEffect(() => {
        if (visible) {
            Animated.timing(slideAnim, {
                toValue: 1,
                duration: 300,
                useNativeDriver: true,
                easing: Easing.out(Easing.poly(4)),
            }).start();

            if (module?.data?.timeline) {
                setTimeline(module.data.timeline);
            } else {
                setTimeline([]);
            }
        } else {
            Animated.timing(slideAnim, {
                toValue: 0,
                duration: 200,
                useNativeDriver: true,
            }).start();
        }
    }, [visible, module]);

    const addStep = (type) => {
        const newStep = {
            id: Date.now().toString(),
            type: type,
            data: type === 'field'
                ? { label: 'New Question', fieldType: 'text' }
                : { content: 'Welcome to the form!' }
        };
        setTimeline([...timeline, newStep]);
    };

    const updateStep = (id, key, value) => {
        setTimeline(timeline.map(step =>
            step.id === id ? { ...step, data: { ...step.data, [key]: value } } : step
        ));
    };

    const removeStep = (id) => {
        Alert.alert(
            "Remove Step",
            "Are you sure you want to remove this step?",
            [
                { text: "Cancel", style: "cancel" },
                {
                    text: "Remove", style: "destructive", onPress: () => {
                        setTimeline(timeline.filter(step => step.id !== id));
                    }
                }
            ]
        );
    };

    const moveStep = (index, direction) => {
        if (direction === 'up' && index > 0) {
            const newTimeline = [...timeline];
            [newTimeline[index], newTimeline[index - 1]] = [newTimeline[index - 1], newTimeline[index]];
            setTimeline(newTimeline);
        } else if (direction === 'down' && index < timeline.length - 1) {
            const newTimeline = [...timeline];
            [newTimeline[index], newTimeline[index + 1]] = [newTimeline[index + 1], newTimeline[index]];
            setTimeline(newTimeline);
        }
    };

    const handlePublish = async () => {
        if (timeline.length === 0) {
            Alert.alert("Warning", "Your form has no steps. Please add at least one.");
            return;
        }

        setLoading(true);
        try {
            const response = await fetch(`${API_URL}/events/${eventId}/modules/${module.id}/publish`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    data: { timeline: timeline }
                })
            });

            if (response.ok) {
                Alert.alert('Success', 'Form configuration published successfully!');
                if (onUpdate) onUpdate();
                handleClose();
            } else {
                Alert.alert('Error', 'Failed to publish configuration.');
            }
        } catch (error) {
            console.error('Error publishing:', error);
            Alert.alert('Error', 'Failed to publish configuration.');
        } finally {
            setLoading(false);
        }
    };

    const fetchTemplates = async () => {
        try {
            const response = await fetch(`${API_URL}/templates`);
            if (response.ok) {
                const data = await response.json();
                setTemplates(data || []);
            }
        } catch (error) {
            console.error('Error fetching templates:', error);
        }
    };

    const handleLoadTemplate = (template) => {
        if (template.data && template.data.timeline) {
            Alert.alert(
                "Load Template",
                "This will replace your current steps. Continue?",
                [
                    { text: "Cancel", style: "cancel" },
                    {
                        text: "Load", onPress: () => {
                            setTimeline(template.data.timeline);
                            setShowTemplateModal(false);
                        }
                    }
                ]
            );
        }
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

    const handleSaveAsTemplate = async () => {
        if (!templateName.trim()) {
            Alert.alert("Error", "Please enter a template name.");
            return;
        }

        try {
            const response = await fetch(`${API_URL}/templates`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: templateName,
                    description: "Created from " + module.name,
                    moduleType: "form",
                    data: { timeline: timeline }
                })
            });
            if (response.ok) {
                Alert.alert("Success", "Template saved!");
                setShowSaveTemplateModal(false);
                setTemplateName('');
            }
        } catch (error) {
            console.error(error);
            Alert.alert("Error", "Failed to save template.");
        }
    };

    const handleClose = () => {
        Animated.timing(slideAnim, {
            toValue: 0,
            duration: 200,
            useNativeDriver: true,
        }).start(() => onClose());
    };

    const renderStep = (step, index) => {
        const isField = step.type === 'field';
        return (
            <View key={step.id} style={styles.stepCard}>
                <View style={[styles.stepHeader, { borderLeftColor: isField ? '#0A84FF' : '#FF9F0A' }]}>
                    <View style={styles.stepTitleContainer}>
                        <View style={[styles.stepIcon, { backgroundColor: isField ? 'rgba(10, 132, 255, 0.1)' : 'rgba(255, 159, 10, 0.1)' }]}>
                            <Ionicons name={isField ? "help-circle" : "chatbubble"} size={18} color={isField ? '#0A84FF' : '#FF9F0A'} />
                        </View>
                        <Text style={styles.stepIndex}>Step {index + 1}</Text>
                        <Text style={styles.stepTypeLabel}>{isField ? "Question" : "Message"}</Text>
                    </View>

                    <View style={styles.stepActions}>
                        <TouchableOpacity style={styles.actionIcon} onPress={() => moveStep(index, 'up')} disabled={index === 0}>
                            <Ionicons name="arrow-up" size={16} color={index === 0 ? '#333' : '#888'} />
                        </TouchableOpacity>
                        <TouchableOpacity style={styles.actionIcon} onPress={() => moveStep(index, 'down')} disabled={index === timeline.length - 1}>
                            <Ionicons name="arrow-down" size={16} color={index === timeline.length - 1 ? '#333' : '#888'} />
                        </TouchableOpacity>
                        <TouchableOpacity style={[styles.actionIcon, styles.deleteBtn]} onPress={() => removeStep(step.id)}>
                            <Ionicons name="trash-outline" size={16} color="#FF453A" />
                        </TouchableOpacity>
                    </View>
                </View>

                <View style={styles.stepBody}>
                    {isField ? (
                        <>
                            <Text style={styles.label}>Question / Label</Text>
                            <TextInput
                                style={styles.input}
                                value={step.data.label}
                                onChangeText={(text) => updateStep(step.id, 'label', text)}
                                placeholder="e.g. What is your name?"
                                placeholderTextColor="#555"
                            />

                            <Text style={styles.label}>Response Type</Text>
                            <View style={styles.typeSelector}>
                                {['text', 'number', 'date'].map(type => (
                                    <TouchableOpacity
                                        key={type}
                                        style={[styles.typeOption, step.data.fieldType === type && styles.typeOptionActive]}
                                        onPress={() => updateStep(step.id, 'fieldType', type)}
                                    >
                                        <Text style={[styles.typeOptionText, step.data.fieldType === type && styles.typeOptionTextActive]}>
                                            {type.charAt(0).toUpperCase() + type.slice(1)}
                                        </Text>
                                    </TouchableOpacity>
                                ))}
                            </View>
                        </>
                    ) : (
                        <>
                            <Text style={styles.label}>Message Content</Text>
                            <TextInput
                                style={[styles.input, styles.textArea]}
                                value={step.data.content}
                                onChangeText={(text) => updateStep(step.id, 'content', text)}
                                placeholder="Enter your message here..."
                                placeholderTextColor="#555"
                                multiline
                            />
                        </>
                    )}
                </View>
            </View>
        );
    };

    if (!visible) return null;

    return (
        <Modal visible={visible} transparent animationType="none" onRequestClose={handleClose}>
            <View style={styles.modalOverlay}>
                <Animated.View style={[styles.modalContainer, {
                    transform: [{
                        translateY: slideAnim.interpolate({
                            inputRange: [0, 1],
                            outputRange: [600, 0] // Slide up from bottom
                        })
                    }]
                }]}>
                    {/* Header */}
                    <View style={styles.header}>
                        <View>
                            <Text style={styles.title}>Form Editor</Text>
                            <Text style={styles.subtitle}>{module?.name || 'Untitled Form'}</Text>
                        </View>
                        <View style={styles.headerButtons}>
                            <TouchableOpacity style={styles.iconButton} onPress={() => {
                                fetchTemplates();
                                setShowTemplateModal(true);
                            }}>
                                <Ionicons name="cloud-download-outline" size={24} color="#FFF" />
                            </TouchableOpacity>
                            <TouchableOpacity style={styles.iconButton} onPress={() => setShowSaveTemplateModal(true)}>
                                <Ionicons name="save-outline" size={24} color="#FFF" />
                            </TouchableOpacity>
                            <TouchableOpacity style={styles.closeButton} onPress={handleClose}>
                                <Ionicons name="close" size={24} color="#FFF" />
                            </TouchableOpacity>
                        </View>
                    </View>

                    {/* Content */}
                    <View style={styles.content}>
                        <ScrollView contentContainerStyle={styles.scrollContent}>
                            {timeline.length === 0 ? (
                                <View style={styles.emptyState}>
                                    <Ionicons name="create-outline" size={48} color="#333" />
                                    <Text style={styles.emptyText}>Start by adding a question or message</Text>
                                </View>
                            ) : (
                                timeline.map((step, index) => renderStep(step, index))
                            )}
                            <View style={{ height: 100 }} />
                        </ScrollView>

                        {/* Floating Action Bar */}
                        <View style={styles.actionBar}>
                            <View style={styles.addButtons}>
                                <TouchableOpacity style={[styles.addBtn, styles.addQuestionBtn]} onPress={() => addStep('field')}>
                                    <Ionicons name="add" size={20} color="#FFF" />
                                    <Text style={styles.addBtnText}>Question</Text>
                                </TouchableOpacity>
                                <TouchableOpacity style={[styles.addBtn, styles.addMessageBtn]} onPress={() => addStep('message')}>
                                    <Ionicons name="chatbubble-outline" size={18} color="#FFF" />
                                    <Text style={styles.addBtnText}>Message</Text>
                                </TouchableOpacity>
                            </View>

                            <TouchableOpacity style={styles.publishBtn} onPress={handlePublish} disabled={loading}>
                                <Text style={styles.publishBtnText}>{loading ? 'Saving...' : 'Save Changes'}</Text>
                            </TouchableOpacity>
                        </View>
                    </View>

                    {/* Template Load Modal */}
                    <Modal visible={showTemplateModal} transparent animationType="fade">
                        <View style={styles.dialogOverlay}>
                            <View style={styles.dialog}>
                                <Text style={styles.dialogTitle}>Start from Template</Text>
                                <ScrollView style={{ maxHeight: 300 }}>
                                    {templates.map(t => (
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
                                    {templates.length === 0 && <Text style={{ color: '#666', textAlign: 'center', padding: 20 }}>No templates found.</Text>}
                                </ScrollView>
                                <TouchableOpacity style={styles.dialogCloseBtn} onPress={() => setShowTemplateModal(false)}>
                                    <Text style={styles.dialogCloseText}>Cancel</Text>
                                </TouchableOpacity>
                            </View>
                        </View>
                    </Modal>

                    {/* Save Template Modal */}
                    <Modal visible={showSaveTemplateModal} transparent animationType="fade">
                        <View style={styles.dialogOverlay}>
                            <View style={styles.dialog}>
                                <Text style={styles.dialogTitle}>Save as Template</Text>
                                <TextInput
                                    style={styles.dialogInput}
                                    placeholder="Template Name"
                                    placeholderTextColor="#666"
                                    value={templateName}
                                    onChangeText={setTemplateName}
                                />
                                <View style={styles.dialogButtons}>
                                    <TouchableOpacity style={[styles.dialogBtn, { backgroundColor: '#333' }]} onPress={() => setShowSaveTemplateModal(false)}>
                                        <Text style={{ color: '#FFF' }}>Cancel</Text>
                                    </TouchableOpacity>
                                    <TouchableOpacity style={[styles.dialogBtn, { backgroundColor: '#0A84FF' }]} onPress={handleSaveAsTemplate}>
                                        <Text style={{ color: '#FFF' }}>Save</Text>
                                    </TouchableOpacity>
                                </View>
                            </View>
                        </View>
                    </Modal>

                </Animated.View>
            </View>
        </Modal>
    );
}

const styles = StyleSheet.create({
    modalOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.5)',
        justifyContent: 'flex-end',
    },
    modalContainer: {
        backgroundColor: '#1C1C1E', // Slightly lighter dark background
        height: '92%',
        borderTopLeftRadius: 20,
        borderTopRightRadius: 20,
        shadowColor: "#000",
        shadowOffset: { width: 0, height: -2 },
        shadowOpacity: 0.3,
        shadowRadius: 10,
        elevation: 10,
        display: 'flex',
        flexDirection: 'column',
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 20,
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2E',
    },
    title: {
        fontSize: 22,
        fontWeight: 'bold',
        color: '#FFF',
    },
    subtitle: {
        fontSize: 14,
        color: '#8E8E93',
        marginTop: 2,
    },
    headerButtons: {
        flexDirection: 'row',
        gap: 16,
    },
    iconButton: {
        padding: 8,
        borderRadius: 8,
        backgroundColor: '#2C2C2E',
    },
    closeButton: {
        padding: 8,
        borderRadius: 8,
        backgroundColor: '#3A3A3C',
        marginLeft: 8,
    },
    content: {
        flex: 1,
        position: 'relative',
    },
    scrollContent: {
        padding: 20,
        paddingBottom: 120, // Space for footer
    },
    emptyState: {
        alignItems: 'center',
        justifyContent: 'center',
        paddingVertical: 100,
        opacity: 0.5,
    },
    emptyText: {
        color: '#8E8E93',
        marginTop: 16,
        fontSize: 16,
    },
    stepCard: {
        backgroundColor: '#2C2C2E',
        borderRadius: 16,
        marginBottom: 16,
        overflow: 'hidden',
        borderWidth: 1,
        borderColor: '#3A3A3C',
    },
    stepHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 16,
        backgroundColor: '#252527',
        borderLeftWidth: 4,
    },
    stepTitleContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 12,
    },
    stepIcon: {
        width: 32,
        height: 32,
        borderRadius: 16,
        alignItems: 'center',
        justifyContent: 'center',
    },
    stepIndex: {
        fontSize: 12,
        color: '#8E8E93',
        fontWeight: '600',
        textTransform: 'uppercase',
    },
    stepTypeLabel: {
        fontSize: 16,
        fontWeight: '600',
        color: '#FFF',
    },
    stepActions: {
        flexDirection: 'row',
        gap: 8,
    },
    actionIcon: {
        padding: 8,
        borderRadius: 6,
        backgroundColor: '#333',
    },
    deleteBtn: {
        backgroundColor: 'rgba(255, 69, 58, 0.1)',
    },
    stepBody: {
        padding: 16,
    },
    label: {
        fontSize: 12,
        color: '#8E8E93',
        marginBottom: 8,
        textTransform: 'uppercase',
        fontWeight: '600',
    },
    input: {
        backgroundColor: '#1C1C1E',
        borderRadius: 8,
        padding: 12,
        fontSize: 16,
        color: '#FFF',
        borderWidth: 1,
        borderColor: '#3A3A3C',
        marginBottom: 16,
    },
    textArea: {
        minHeight: 80,
        textAlignVertical: 'top',
    },
    typeSelector: {
        flexDirection: 'row',
        backgroundColor: '#1C1C1E',
        borderRadius: 8,
        padding: 4,
        borderWidth: 1,
        borderColor: '#3A3A3C',
    },
    typeOption: {
        flex: 1,
        paddingVertical: 8,
        alignItems: 'center',
        borderRadius: 6,
    },
    typeOptionActive: {
        backgroundColor: '#3A3A3C',
    },
    typeOptionText: {
        fontSize: 14,
        color: '#8E8E93',
        fontWeight: '500',
    },
    typeOptionTextActive: {
        color: '#FFF',
        fontWeight: '600',
    },
    actionBar: {
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        backgroundColor: '#1C1C1E', // or #252527
        borderTopWidth: 1,
        borderTopColor: '#2C2C2E',
        padding: 16,
        paddingBottom: Platform.OS === 'ios' ? 32 : 16,
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    addButtons: {
        flexDirection: 'row',
        gap: 12,
    },
    addBtn: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingVertical: 10,
        paddingHorizontal: 16,
        borderRadius: 10,
        gap: 6,
    },
    addQuestionBtn: {
        backgroundColor: '#0A84FF',
    },
    addMessageBtn: {
        backgroundColor: '#FF9F0A',
    },
    addBtnText: {
        color: '#FFF',
        fontWeight: '600',
        fontSize: 14,
    },
    publishBtn: {
        backgroundColor: '#30D158',
        paddingVertical: 12,
        paddingHorizontal: 24,
        borderRadius: 10,
    },
    publishBtnText: {
        color: '#FFF',
        fontWeight: 'bold',
        fontSize: 15,
    },
    dialogOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.7)',
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
    },
    dialog: {
        backgroundColor: '#1C1C1E',
        borderRadius: 16,
        padding: 24,
        width: '100%',
        maxWidth: 400,
        borderWidth: 1,
        borderColor: '#3A3A3C',
    },
    dialogTitle: {
        fontSize: 20,
        fontWeight: 'bold',
        color: '#FFF',
        marginBottom: 16,
    },
    templateItemWrapper: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: '#2C2C2E',
        borderRadius: 8,
        marginBottom: 8,
        overflow: 'hidden',
    },
    templateItem: {
        flex: 1,
        padding: 12,
    },
    deleteTemplateBtn: {
        padding: 12,
        backgroundColor: 'rgba(255, 69, 58, 0.1)',
        height: '100%',
        justifyContent: 'center',
        alignItems: 'center',
    },
    templateItemName: {
        fontSize: 16,
        color: '#FFF',
        fontWeight: '600',
    },
    templateItemDesc: {
        fontSize: 14,
        color: '#8E8E93',
        marginTop: 4,
    },
    dialogCloseBtn: {
        marginTop: 16,
        padding: 12,
        alignItems: 'center',
        borderRadius: 8,
        backgroundColor: '#3A3A3C',
    },
    dialogCloseText: {
        color: '#FFF',
        fontWeight: '600',
    },
    dialogInput: {
        backgroundColor: '#2C2C2E',
        color: '#FFF',
        padding: 12,
        borderRadius: 8,
        marginBottom: 16,
        borderWidth: 1,
        borderColor: '#3A3A3C',
    },
    dialogButtons: {
        flexDirection: 'row',
        gap: 12,
        justifyContent: 'flex-end',
    },
    dialogBtn: {
        paddingVertical: 10,
        paddingHorizontal: 16,
        borderRadius: 8,
    },
});
